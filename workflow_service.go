package endly

import (
	"fmt"
	"errors"
	"github.com/viant/endly/common"
	"github.com/viant/toolbox"
	"strings"
)

const WorkflowServiceId = "workflow"

type WorkflowRunRequest struct {
	Name   string
	Params map[string]interface{}
	Tasks  map[string]string
}

type RunWorkflowResponse struct {
	Name            string
	Data            map[string]interface{}
	SessionInfo     *SessionInfo
	TasksActivities []*WorkflowTaskActivity
}

type WorkflowTaskActivity struct {
	Task              string
	ServiceActivities []*WorkflowServiceActivity
	Data              map[string]interface{}
}

type WorkflowServiceActivity struct {
	Service         string
	Action          string
	ServiceRequest  interface{}
	ServiceResponse interface{}
}

type WorkflowRegisterRequest struct {
	Workflow *Workflow
}

type WorkflowLoadRequest struct {
	Source *Resource
}

type WorkflowService struct {
	*AbstractService
	dao      *WorkflowDao
	registry map[string]*Workflow
}

func (s *WorkflowService) Register(workflow *Workflow) error {
	err := workflow.Validate()
	if err != nil {
		return err
	}
	s.registry[workflow.Name] = workflow
	return nil
}

func (s *WorkflowService) Workflow(name string) (*Workflow, error) {
	if result, found := s.registry[name]; found {
		return result, nil
	}
	return nil, fmt.Errorf("Failed to lookup workflow: %v", name)
}

func (s *WorkflowService) runWorkflow(context *Context, request *WorkflowRunRequest) (*RunWorkflowResponse, error) {
	workflow, err := s.Workflow(request.Name)
	if err != nil {
		return nil, err
	}
	var state = context.State()
	var response = &RunWorkflowResponse{
		TasksActivities: make([]*WorkflowTaskActivity, 0),
		Data:            state,
	}
	var params = common.NewMap()
	if len(request.Params) > 0 {
		for k, v := range request.Params {
			if toolbox.IsString(v) {
				params[k] = context.Expand(toolbox.AsString(v))
			} else {
				params[k] = v
			}
		}
	}
	state.Put("workflow", workflow.Name)
	state.Put("params", request.Params)
	workflow.Variables.Eval(state)

	var hasAllowedTasks = len(request.Tasks) > 0

	for _, task := range workflow.Tasks {
		var allowedServiceActions map[int]bool

		if hasAllowedTasks {
			allowedActionIndexes, ok := request.Tasks[task.Name];
			if ! ok {
				continue
			}
			allowedServiceActions = make(map[int]bool)
			for _, index := range strings.Split(allowedActionIndexes, ",") {
				if index == "" {
					continue
				}
				allowedServiceActions[toolbox.AsInt(index)] = true
			}
		}
		var hasAllowedActions = len(allowedServiceActions) > 0

		state.Put("task", task.Name)
		task.Variables.Eval(state)
		var taskActivity = &WorkflowTaskActivity{
			ServiceActivities: make([]*WorkflowServiceActivity, 0),
			Data:              make(map[string]interface{}),
		}
		response.TasksActivities = append(response.TasksActivities, taskActivity)
		for i, action := range task.Actions {
			if hasAllowedActions && ! allowedServiceActions[i] {
				continue
			}
			state.Put("service", action.Service)
			state.Put("action", action.Action)
			action.Variables.Eval(state)
			action.Variables.Apply(common.Map(taskActivity.Data), state, true)
			service, err := context.Service(action.Service)
			if err != nil {
				return nil, err
			}
			serviceRequest, err := service.NewRequest(action.Action)
			if err != nil {
				return nil, err
			}

			serviceActivity := &WorkflowServiceActivity{
				ServiceRequest: serviceRequest,
			}
			taskActivity.ServiceActivities = append(taskActivity.ServiceActivities, serviceActivity)

			err = converter.AssignConverted(serviceRequest, action.Request)
			if err != nil {
				return response, err
			}
			serviceResponse := service.Run(context, serviceRequest)
			serviceActivity.ServiceResponse = serviceResponse
			if serviceResponse.Error != "" {
				return nil, errors.New(serviceResponse.Error)
			}
			var responseMap = make(map[string]interface{})
			if serviceResponse.Response != nil {
				converter.AssignConverted(responseMap, serviceResponse.Response)
				action.Variables.Apply(common.Map(responseMap), common.Map(taskActivity.Data), false)
			}
		}
		task.Variables.Apply(common.Map(taskActivity.Data), common.Map(response.Data), false)
	}
	return response, nil
}

func (s *WorkflowService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{Status: "ok"}
	var err error
	switch actualRequest := request.(type) {
	case *WorkflowRunRequest:
		response.Response, err = s.runWorkflow(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run workflow: %v, %v", actualRequest.Name, err)
		}
	case *WorkflowRegisterRequest:
		err := s.Register(actualRequest.Workflow)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to register workflow: %v, %v", actualRequest.Workflow.Name, err)
		}
	case *WorkflowLoadRequest:
		workflow, err := s.dao.Load(context, actualRequest.Source)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to load workflow: %v, %v", actualRequest.Source, err)
		}
		err = s.Register(workflow)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to register workflow: %v, %v", actualRequest.Source, err)
		}
	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *WorkflowService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "run":
		return &WorkflowRunRequest{}, nil
	case "register":
		return &WorkflowRegisterRequest{}, nil

	case "load":
		return &WorkflowLoadRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func NewWorkflowService() Service {
	var result = &WorkflowService{
		AbstractService: NewAbstractService(WorkflowServiceId),
		dao:             NewWorkflowDao(),
	}
	result.AbstractService.Service = result
	return result
}