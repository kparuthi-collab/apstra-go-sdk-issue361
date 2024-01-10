package apstra

import (
	"context"
	"fmt"
	"github.com/orsinium-labs/enum"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	portAny       = "any"
	portRangeSep  = "-"
	portRangesSep = ","
)

// RULE_SCHEMA = {
//    'id': s.Optional(s.NodeId(description='ID of the rule node')),
//    'label': s.GenericName(description='Unique user-friendly name of the rule'),
//    'protocol': s.SecurityRuleProtocol(),
//    'src_port': s.PortSetOrAny(),
//    'dst_port': s.PortSetOrAny(),
//    'description': s.Optional(s.Description(), load_default=''),
//    'action': s.SecurityRuleAction() //             ['deny', 'deny_log', 'permit', 'permit_log'],
//}

type PolicyRuleAction enum.Member[string]

var (
	PolicyRuleActionDeny      = PolicyRuleAction{Value: "deny"}
	PolicyRuleActionDenyLog   = PolicyRuleAction{Value: "deny_log"}
	PolicyRuleActionPermit    = PolicyRuleAction{Value: "permit"}
	PolicyRuleActionPermitLog = PolicyRuleAction{Value: "permit_log"}
	PolicyRuleActions         = enum.New(PolicyRuleActionDeny, PolicyRuleActionDenyLog, PolicyRuleActionPermit, PolicyRuleActionPermitLog)
)

type PortRange struct {
	first uint16
	last  uint16
}

func (o PortRange) string() string {
	switch {
	case o.first == o.last:
		return strconv.Itoa(int(o.first))
	case o.first < o.last:
		return strconv.Itoa(int(o.first)) + portRangeSep + strconv.Itoa(int(o.last))
	default:
		return strconv.Itoa(int(o.last)) + portRangeSep + strconv.Itoa(int(o.first))
	}
}

type rawPortRanges string

func (o rawPortRanges) parse() (PortRanges, error) {
	if o == portAny {
		return []PortRange{}, nil
	}

	rawRangeSlice := strings.Split(string(o), portRangesSep)
	result := make([]PortRange, len(rawRangeSlice))
	for i, raw := range rawRangeSlice {
		var first, last uint64
		var err error
		portStrs := strings.Split(raw, portRangeSep)
		switch len(portStrs) {
		case 1:
			first, err = strconv.ParseUint(raw, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("error parsing port range '%s' - %w", raw, err)
			}
			last = first
		case 2:
			first, err = strconv.ParseUint(portStrs[0], 10, 16)
			if err != nil {
				return nil, fmt.Errorf("error parsing first element of port range '%s' - %w", raw, err)
			}
			last, err = strconv.ParseUint(portStrs[1], 10, 16)
			if err != nil {
				return nil, fmt.Errorf("error parsing last element of port range '%s' - %w", raw, err)
			}
		default:
			return nil, fmt.Errorf("cannot parse port range '%s'", raw)
		}
		if first > math.MaxUint16 || last > math.MaxUint16 {
			return nil, fmt.Errorf("port spec '%s' falls outside of range %d-%d", raw, 0, math.MaxUint16)
		}
		result[i] = PortRange{
			first: uint16(first),
			last:  uint16(last),
		}
	}
	return result, nil
}

type PortRanges []PortRange

func (o PortRanges) string() string {
	if len(o) == 0 {
		return portAny
	}
	sb := strings.Builder{}
	sb.WriteString(o[0].string())
	for _, pr := range o[1:] {
		sb.WriteString(portRangesSep + pr.string())
	}
	return sb.String()
}

type PolicyRule struct {
	Id          ObjectId
	Label       string
	Description string
	Protocol    string
	Action      PolicyRuleAction
	SrcPort     PortRanges
	DstPort     PortRanges
}

func (o PolicyRule) raw() *rawPolicyRule {
	return &rawPolicyRule{
		Id:          o.Id,
		Label:       o.Label,
		Description: o.Description,
		Protocol:    o.Protocol,
		Action:      o.Action.Value,
		SrcPort:     rawPortRanges(o.SrcPort.string()),
		DstPort:     rawPortRanges(o.DstPort.string()),
	}
}

type rawPolicyRule struct {
	Id          ObjectId      `json:"id,omitempty"`
	Label       string        `json:"label"`
	Description string        `json:"description"`
	Protocol    string        `json:"protocol"`
	Action      string        `json:"action"`
	SrcPort     rawPortRanges `json:"src_port"`
	DstPort     rawPortRanges `json:"dst_port"`
}

func (o rawPolicyRule) polish() (*PolicyRule, error) {
	action := PolicyRuleActions.Parse(o.Action)
	if action == nil {
		return nil, fmt.Errorf("unknown policy rule action %q", o.Action)
	}
	srcPort, err := o.SrcPort.parse()
	if err != nil {
		return nil, err
	}
	dstPort, err := o.DstPort.parse()
	if err != nil {
		return nil, err
	}
	return &PolicyRule{
		Id:          o.Id,
		Label:       o.Label,
		Description: o.Description,
		Protocol:    o.Protocol,
		Action:      *action,
		SrcPort:     srcPort,
		DstPort:     dstPort,
	}, nil
}

func (o *TwoStageL3ClosClient) getPolicyRuleIdByLabel(ctx context.Context, policyId ObjectId, label string) (ObjectId, error) {
	start := time.Now()
	for i := 0; i <= dcClientMaxRetries; i++ {
		time.Sleep(dcClientRetryBackoff * time.Duration(i))
		policy, err := o.getPolicy(ctx, policyId)
		if err != nil {
			return "", err
		}
		for _, rule := range policy.Rules {
			if rule.Label == label {
				return rule.Id, nil
			}
		}
	}
	return "", fmt.Errorf("rule '%s' didn't appear in policy '%s' after %s", label, policyId, time.Since(start))
}

func (o *TwoStageL3ClosClient) addPolicyRule(ctx context.Context, rule *rawPolicyRule, position int, policyId ObjectId) (ObjectId, error) {
	// ensure exclusive access to the policy while we recalculate the rules
	lockId := o.lockId(policyId)
	o.client.lock(lockId)
	defer o.client.unlock(lockId)

	policy, err := o.getPolicy(ctx, policyId)
	if err != nil {
		return "", err
	}

	currentRuleCount := len(policy.Rules)

	if position < 0 {
		position = currentRuleCount
	}

	switch {
	case currentRuleCount == 0:
		// empty rule set is an easy case
		policy.Rules = []rawPolicyRule{*rule}
	case position == 0:
		// insert at the beginning
		policy.Rules = append([]rawPolicyRule{*rule}, policy.Rules...)
	case position >= currentRuleCount:
		// insert at the end
		policy.Rules = append(policy.Rules, *rule)
	default:
		// insert somewhere in the middle
		policy.Rules = append(policy.Rules[:position+1], policy.Rules[position:]...)
		policy.Rules[position] = *rule
	}

	// push the new policy
	err = o.updatePolicy(ctx, policyId, policy.request())
	if err != nil {
		return "", err
	}

	return o.getPolicyRuleIdByLabel(ctx, policyId, rule.Label)
}

func (o *TwoStageL3ClosClient) deletePolicyRuleById(ctx context.Context, policyId ObjectId, ruleId ObjectId) error {
	// ensure exclusive access to the policy while we recalculate the rules
	lockId := o.lockId(policyId)
	o.client.lock(lockId)
	defer o.client.unlock(lockId)

	policy, err := o.getPolicy(ctx, policyId)
	if err != nil {
		return err
	}

	ruleIdx := -1
	for i, rule := range policy.Rules {
		if rule.Id == ruleId {
			ruleIdx = i
			break
		}
	}

	if ruleIdx < 0 {
		return ClientErr{
			errType: ErrNotfound,
			err:     fmt.Errorf("rule id '%s' not found in policy '%s'", ruleId, policyId),
		}
	}

	policy.Rules = append(policy.Rules[:ruleIdx], policy.Rules[ruleIdx+1:]...)
	return o.updatePolicy(ctx, policyId, policy.request())
}
