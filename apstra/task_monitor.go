package apstra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	apiUrlTasksPrefix = apiUrlBlueprintsPrefix
	apiUrlTasksSuffix = "/tasks/"

	taskMonFirstCheckDelay = 100 * time.Millisecond
	taskMonPollInterval    = 500 * time.Millisecond

	taskStatusOngoing = "in_progress"
	taskStatusInit    = "init"
	taskStatusFail    = "failed"
	taskStatusSuccess = "succeeded"
	taskStatusTimeout = "timeout"
)

// getAllTasksResponse is sent by Apstra in response to GET at
// 'apiUrlTaskPrefix + blueprintId + apiUrlTaskSuffix'
type getAllTasksResponse struct {
	Items []struct {
		Status      string `json:"status"`
		BeginAt     string `json:"begin_at"`
		RequestData struct {
			Url    string `json:"url"`
			Method string `json:"method"`
		} `json:"request_data"`
		UserId              string `json:"user_id"`
		LastUpdatedAt       string `json:"last_updated_at"`
		UserName            string `json:"user_name"`
		CreatedAt           string `json:"created_at"`
		ConfigLastUpdatedAt string `json:"config_last_updated_at"`
		UserIp              string `json:"user_ip"`
		Type                string `json:"type"`
		Id                  TaskId `json:"id"`
	} `json:"items"`
}

type detailedStatus struct {
	ApiResponse            json.RawMessage `json:"api_response"`
	ConfigBlueprintVersion int             `json:"config_blueprint_version"`
	Errors                 json.RawMessage `json:"errors"`
	ErrorCode              int             `json:"error_code"`
}

// getTaskResponse is sent by Apstra in response to GET at
// 'apiUrlTaskPrefix + blueprintId + apiUrlTaskSuffix + taskId'
type getTaskResponse struct {
	Status      string `json:"status"`
	BeginAt     string `json:"begin_at"`
	RequestData struct {
		Url     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Args    map[string]string `json:"args"`
		Data    json.RawMessage   `json:"data"`
		Method  string            `json:"method"`
	} `json:"request_data"`
	UserId              string         `json:"user_id"`
	LastUpdatedAt       string         `json:"last_updated_at"`
	UserName            string         `json:"user_name"`
	CreatedAt           string         `json:"created_at"`
	DetailedStatus      detailedStatus `json:"detailed_status"`
	ConfigLastUpdatedAt string         `json:"config_last_updated_at"`
	UserIp              string         `json:"user_ip"`
	Type                string         `json:"type"`
	Id                  TaskId         `json:"id"`
}

// taskMonitorMonReq uniquely identifies an Apstra task which can be tracked at
// /api/blueprint/<id>/tasks and /api/blueprint/<id>/tasks/<id> API endpoints.
// This structure is submitted by a caller via taskMonitor's taskInChan. When
// the task is no longer outstanding (success, timeout, failed), taskMonitor
// responds via responseChan with the complete getTaskResponse structure received
// from Apstra and an error, if appropriate.
type taskMonitorMonReq struct {
	bluePrintId  ObjectId                 // task API calls must reference a blueprint
	taskId       TaskId                   // tracks the task
	responseChan chan<- *taskCompleteInfo // talk here when the task is complete
}

// taskCompleteInfo is generated by taskMonitor when a TaskId exits pending modes
// ("init" or "in_progress"). The id field represents the object created by
// the task, and result can be any status representing a completed task
// ("succeeded", "failed", "timeout")
type taskCompleteInfo struct {
	status *getTaskResponse
	err    error
}

// pendingTaskData is a map keyed by blueprintId (ObjectId). Values are maps of TaskId
// to chan<- *taskCompleteInfo (callers expect API response here).
// So, it looks like this:
//
//	pendingTaskData{
//		ObjectId("blueprint_1"): {
//			TaskId("task_abc"): make(chan<- *taskCompleteInfo),
//			TaskId("task_def"): make(chan<- *taskCompleteInfo),
//		},
//		ObjectId("blueprint_2"): {
//			TaskId("task_uvw"): make(chan<- *taskCompleteInfo),
//			TaskId("task_xyz"): make(chan<- *taskCompleteInfo),
//		},
//	}
type pendingTaskData map[ObjectId]map[TaskId]chan<- *taskCompleteInfo

func (o pendingTaskData) add(in *taskMonitorMonReq) {
	if _, found := o[in.bluePrintId]; !found {
		// blueprint not found in pendingTaskData - create that blueprint's task map
		o[in.bluePrintId] = make(map[TaskId]chan<- *taskCompleteInfo)
	}
	o[in.bluePrintId][in.taskId] = in.responseChan
}

func (o pendingTaskData) del(bpId ObjectId, taskId TaskId) {
	if _, found := o[bpId]; !found {
		// blueprint not found; nothing to delete; we're done here
		return
	}
	// delete the task from the blueprint->task map
	delete(o[bpId], taskId)

	if len(o[bpId]) == 0 {
		// delete the blueprint from the pendingTaskData map
		delete(o, bpId)
	}
}

func (o pendingTaskData) blueprintCount() int {
	return len(o)
}

func (o pendingTaskData) isEmpty() bool {
	return o.blueprintCount() == 0
}

func (o pendingTaskData) taskListByBlueprint(bpId ObjectId) []TaskId {
	taskMap := o[bpId]
	result := make([]TaskId, len(taskMap))
	i := 0
	for taskId := range taskMap {
		result[i] = taskId
		i++
	}
	return result
}

// a taskMonitor runs as an independent goroutine, accepts task{}s to monitor
// via taskInChan, closes the task's `done` channel when it detects apstra
// has completed the task.
type taskMonitor struct {
	client            *Client                   // for making Apstra API calls
	taskInChan        <-chan *taskMonitorMonReq // for learning about new tasks
	timer             *time.Timer               // triggers check()
	errChan           chan<- error              // error feedback to main loop
	lock              sync.Mutex                // control access to mapBpIdToTask
	tmQuit            <-chan struct{}           // taskMonitor initiates shutdown when this closes
	pendingTaskData   pendingTaskData           // data structure containing monitored task info
	shutdownRequested bool                      // flag indicating we should exit
	timerSetTime      time.Time                 // used to calculate time delay for logs (probably remove)
}

// newTaskMonitor creates a new taskMonitor, but does not start it.
func newTaskMonitor(c *Client) *taskMonitor {
	monitor := taskMonitor{
		timer:           time.NewTimer(0), // write dummy event to channel immediately
		client:          c,
		taskInChan:      c.taskMonChan,
		errChan:         c.cfg.ErrChan,
		pendingTaskData: make(pendingTaskData),
	}
	<-monitor.timer.C // read dummy event to clear timer channel
	return &monitor
}

// start causes the taskMonitor to run
func (o *taskMonitor) start() {
	o.tmQuit = o.client.tmQuit
	go o.run()
}

// run is the main taskMonitor loop
func (o *taskMonitor) run() {
	// main loop
	for {
		if o.tmShouldExit() {
			return
		}
		select {
		// timer event
		case <-o.timer.C:
			o.client.Logf(3, "task timer fired after %s", time.Since(o.timerSetTime).String())
			go o.check()
		case in := <-o.taskInChan: // new task event
			o.client.Log(2, fmt.Sprintf("new task arrived: bp '%s', task '%s'", in.bluePrintId, in.taskId))
			o.stopTimer()
			o.acquireLock("tm main loop")
			o.pendingTaskData.add(in)
			o.releaseLock("tm main loop")
			o.startTimer()
		case <-o.tmQuit: // program exit
			o.client.Log(1, "task monitor exiting")
			o.shutdownRequested = true
		}
	}
}

// ShouldExit returns true when shutdown has been requested
// and the task monitor queue is empty
func (o *taskMonitor) tmShouldExit() bool {
	return o.shutdownRequested && o.pendingTaskData.isEmpty()
}

// stopTimer stops the timer and drains the timer channel
func (o *taskMonitor) stopTimer() {
	o.timer.Stop()
	select {
	case <-o.timer.C:
		o.client.Log(1, "snagged a stray timer event after stopping the timer")
	default:
		o.client.Log(1, "timer stopped, channel empty")
	}
}

// startTimer starts the timer
func (o *taskMonitor) startTimer() {
	o.timerSetTime = time.Now()
	o.timer.Reset(taskMonFirstCheckDelay)
}

func (o *taskMonitor) acquireLock(who string) {
	o.client.Logf(3, "%s locking task monitor", who)
	o.lock.Lock()
	o.client.Logf(3, "%s locked task monitor", who)
}

func (o *taskMonitor) releaseLock(who string) {
	o.client.Logf(3, "%s unlocking task monitor", who)
	o.lock.Unlock()
	o.client.Logf(3, "%s unlocked task monitor", who)
}

func (o *taskMonitor) checkBlueprints() {
	// loop over blueprints known to have outstanding tasks
	for bpId := range o.pendingTaskData {
		taskIdList := o.pendingTaskData.taskListByBlueprint(bpId)
		// get task result info from Apstra
		taskIdToStatus, err := o.client.getBlueprintTasksStatus(o.client.ctx, bpId, taskIdList)
		if err != nil {
			o.handleErr(fmt.Errorf("error getting tasks for blueprint %s - %w", bpId, err))
			continue
		}
		o.checkTasksInBlueprint(bpId, taskIdToStatus)
	}
}

func (o *taskMonitor) handleErr(err error) {
	if o.errChan != nil {
		o.errChan <- err
	} else {
		o.client.Log(0, err.Error())
	}
}

// checkTasksInBlueprint takes a blueprint ID and a [TaskId]status (string)
// fetched using the task summary API endpoint. We'll ask Apstra for the
// detailed API output for any tasks no longer in progress, and return that via
// the caller specified channel.
func (o *taskMonitor) checkTasksInBlueprint(bpId ObjectId, mapTaskIdToStatus map[TaskId]string) {
	// loop over *all* outstanding tasks associated with this blueprint
	for taskId, responseChan := range o.pendingTaskData[bpId] {
		// make sure Apstra response (input to this function) includes our taskId
		if _, found := mapTaskIdToStatus[taskId]; !found {
			// Apstra response doesn't have our task ID
			//   error, delete it from the slice, next task
			responseChan <- &taskCompleteInfo{
				err: fmt.Errorf("blueprint '%s' task '%s' unknown to Apstra server", bpId, taskId),
			}
			o.pendingTaskData.del(bpId, taskId)
			continue
		}

		// What did Apstra say about our taskId?
		switch mapTaskIdToStatus[taskId] {
		case taskStatusInit:
			continue // still working; skip to next task
		case taskStatusOngoing:
			continue // still working; skip to next task
		case taskStatusFail: // done; fallthrough
		case taskStatusSuccess: // done; fallthrough
		case taskStatusTimeout: // done; fallthrough
		default: // something else?
			// Unexpected task status response from Apstra:
			//   error, delete it from the slice, next task
			responseChan <- &taskCompleteInfo{
				err: fmt.Errorf("blueprint '%s' task '%s' status unexpected: %s", bpId, taskId, mapTaskIdToStatus[taskId]),
			}
			o.pendingTaskData.del(bpId, taskId)
			continue
		}

		// if we got here, we're able to return a recognized and conclusive
		// task status result to the caller. Fetch the full details from Apstra.
		taskInfo, err := o.client.getBlueprintTaskStatusById(o.client.ctx, bpId, taskId)
		responseChan <- &taskCompleteInfo{
			status: taskInfo,
			err:    err,
		}

		// remove this task from the list of monitored tasks
		o.pendingTaskData.del(bpId, taskId)
	}
}

// check
//
//	invokes checkBlueprints
//	          invokes checkTasksInBlueprint
func (o *taskMonitor) check() {
	o.acquireLock("check")
	o.checkBlueprints()
	if !o.pendingTaskData.isEmpty() {
		o.client.Logf(1, "have %d blueprints with outstanding tasks, resetting timer", o.pendingTaskData.blueprintCount())
		o.timer.Reset(taskMonPollInterval)
	}
	o.releaseLock("check")
}

// taskListToFilterExpr returns the supplied []ObjectId as a string prepped for
// Apstra's API response filter. Something like: "id in ['abc','def']"
func taskListToFilterExpr(in []TaskId) string {
	var quotedList []string
	for i := range in {
		quotedList = append(quotedList, "'"+string(in[i])+"'")
	}
	return "id in [" + strings.Join(quotedList, ",") + "]"
}

// getBlueprintTasksStatus returns a map of TaskId to status (strings like
// "succeeded", "init", etc...)
func (o *Client) getBlueprintTasksStatus(ctx context.Context, bpid ObjectId, taskIdList []TaskId) (map[TaskId]string, error) {
	apstraUrl, err := url.Parse(apiUrlTasksPrefix + string(bpid) + apiUrlTasksSuffix)
	apstraUrl.RawQuery = url.Values{"filter": []string{taskListToFilterExpr(taskIdList)}}.Encode()
	if err != nil {
		return nil, fmt.Errorf("error parsing url '%s' - %w",
			apiUrlTasksPrefix+string(bpid)+apiUrlTasksSuffix, err)
	}
	response := &getAllTasksResponse{}
	err = o.talkToApstra(ctx, &talkToApstraIn{
		method:      http.MethodGet,
		url:         apstraUrl,
		apiInput:    nil,
		apiResponse: response,
		doNotLogin:  false,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting getAllTasksResponse for blueprint '%s' - %w", bpid, err)
	}

	result := make(map[TaskId]string)
	for _, i := range response.Items {
		if i.Status == "" {
			return nil, fmt.Errorf("server resopnse included empty task status")
		}
		if i.Id == "" {
			return nil, fmt.Errorf("server resopnse included empty task id")
		}
		result[i.Id] = i.Status
	}
	return result, nil
}

func (o *Client) getBlueprintTaskStatusById(ctx context.Context, bpid ObjectId, tid TaskId) (*getTaskResponse, error) {
	apstraUrl, err := url.Parse(apiUrlTasksPrefix + string(bpid) + apiUrlTasksSuffix + string(tid))
	if err != nil {
		return nil, fmt.Errorf("error parsing url '%s' - %w",
			apiUrlTasksPrefix+string(bpid)+apiUrlTasksSuffix+string(tid), err)
	}
	result := &getTaskResponse{}
	return result, o.talkToApstra(ctx, &talkToApstraIn{
		method:      http.MethodGet,
		url:         apstraUrl,
		apiInput:    nil,
		apiResponse: result,
		doNotLogin:  false,
	})
}

func blueprintIdFromUrl(in *url.URL) ObjectId {
	split1 := strings.Split(in.String(), apiUrlBlueprintsPrefix)
	if len(split1) != 2 {
		return ""
	}

	split2 := strings.Split(split1[1], apiUrlPathDelim)
	if len(split1) == 0 {
		return ""
	}

	return ObjectId(split2[0])
}

// waitForTaskCompletion interacts with the taskMonitor, returns the Apstra API
// *getTaskResponse
func waitForTaskCompletion(bId ObjectId, tId TaskId, mon chan *taskMonitorMonReq) (*getTaskResponse, error) {
	// todo: restore log message below
	//debugStr(1, fmt.Sprintf("awaiting completion of blueprint '%s' task '%s", bId, tId))
	// task status update channel (how we'll learn the task is complete
	reply := make(chan *taskCompleteInfo, 1) // Task Complete Info Channel
	defer close(reply)

	// submit our task to the task monitor
	mon <- &taskMonitorMonReq{
		bluePrintId:  bId,
		taskId:       tId,
		responseChan: reply,
	}

	tci := <-reply
	return tci.status, tci.err

}
