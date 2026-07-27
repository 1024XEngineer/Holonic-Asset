package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/generation/domain"
	"github.com/1024XEngineer/Holonic-Asset/internal/generation/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/generation/handler"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/task/domain"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type requestServiceStub struct {
	createRequest *domain.GenerationRequest
	detail        *domain.GenerationDetail
	cancelErr     error
	confirmErr    error
}

func (s *requestServiceStub) Create(
	_ context.Context,
	request *domain.GenerationRequest,
) (domain.RunID, error) {
	s.createRequest = request
	return 17, nil
}

func (s *requestServiceStub) Get(
	context.Context,
	domain.RunID,
) (*domain.GenerationDetail, error) {
	return s.detail, nil
}

func (s *requestServiceStub) Cancel(context.Context, domain.RunID) error {
	return s.cancelErr
}

func (s *requestServiceStub) ConfirmCandidate(
	context.Context,
	domain.RunID,
	domain.CandidateID,
) error {
	return s.confirmErr
}

func TestCreateMapsTransportRequest(t *testing.T) {
	stub := &requestServiceStub{}
	generationHandler := handler.NewGenerationHandler(stub)
	parameters := json.RawMessage(`{"size":{"width":64,"height":64}}`)
	request := dto.CreateGenerationRequest{
		ProjectID:              2,
		AssetID:                3,
		Kind:                   domain.RequestKindGenerateCharacter,
		Prompt:                 "hero",
		ReferenceMediaIDs:      []string{"media-1"},
		TargetAssetResourceIDs: []uint{4},
		Parameters:             parameters,
	}

	response, err := generationHandler.Create(newHandlerContext(), request)
	if err != nil {
		t.Fatalf("create generation: %v", err)
	}
	if response.GenerationRunID != 17 {
		t.Fatalf("expected run ID 17, got %d", response.GenerationRunID)
	}
	if stub.createRequest == nil {
		t.Fatal("expected generation request")
	}
	if stub.createRequest.ProjectID != request.ProjectID ||
		stub.createRequest.AssetID != request.AssetID ||
		stub.createRequest.Kind != request.Kind ||
		stub.createRequest.Prompt != request.Prompt ||
		!reflect.DeepEqual(stub.createRequest.ReferenceMediaIDs, request.ReferenceMediaIDs) ||
		!reflect.DeepEqual(stub.createRequest.TargetAssetResourceIDs, request.TargetAssetResourceIDs) ||
		!reflect.DeepEqual(stub.createRequest.Parameters, request.Parameters) {
		t.Fatalf("unexpected generation request: %+v", stub.createRequest)
	}
}

func TestGetMapsGenerationDetail(t *testing.T) {
	confirmedCandidateID := domain.CandidateID(9)
	taskStatus := taskdomain.StatusProcessing
	stub := &requestServiceStub{detail: &domain.GenerationDetail{
		Run: domain.GenerationRun{
			ID:                   7,
			ProjectID:            2,
			Lifecycle:            domain.RunLifecycleWaitingConfirmation,
			ConfirmedCandidateID: &confirmedCandidateID,
			Request: domain.GenerationRequest{
				Kind: domain.RequestKindGenerateCharacter,
			},
		},
		Steps: []domain.StepDetail{{
			Step: domain.Step{
				ID:           8,
				Type:         "generate_image",
				Executor:     domain.StepExecutorAI,
				Dependencies: []domain.StepID{6},
			},
			TaskStatus: &taskStatus,
		}},
		Candidates: []domain.Candidate{{
			ID:       9,
			MediaIDs: []string{"media-1"},
		}},
	}}
	generationHandler := handler.NewGenerationHandler(stub)

	response, err := generationHandler.Get(
		newHandlerContext(),
		dto.GetGenerationRequest{GenerationRunID: 7},
	)
	if err != nil {
		t.Fatalf("get generation: %v", err)
	}
	if response.ID != 7 || response.ProjectID != 2 ||
		response.Kind != domain.RequestKindGenerateCharacter ||
		response.Lifecycle != domain.RunLifecycleWaitingConfirmation {
		t.Fatalf("unexpected run response: %+v", response)
	}
	if len(response.Steps) != 1 || response.Steps[0].ID != 8 ||
		response.Steps[0].TaskStatus == nil ||
		*response.Steps[0].TaskStatus != taskdomain.StatusProcessing {
		t.Fatalf("unexpected steps response: %+v", response.Steps)
	}
	if len(response.Candidates) != 1 || response.Candidates[0].ID != 9 ||
		response.ConfirmedCandidateID == nil || *response.ConfirmedCandidateID != 9 {
		t.Fatalf("unexpected candidates response: %+v", response.Candidates)
	}
}

func TestCommandResponsesReflectServiceResult(t *testing.T) {
	wantErr := errors.New("command failed")

	tests := []struct {
		name    string
		stub    *requestServiceStub
		invoke  func(*handler.GenerationHandler) (bool, error)
		want    bool
		wantErr error
	}{
		{
			name: "cancel success",
			stub: &requestServiceStub{},
			invoke: func(h *handler.GenerationHandler) (bool, error) {
				response, err := h.Cancel(newHandlerContext(), dto.CancelGenerationRequest{GenerationRunID: 7})
				return response.Cancelled, err
			},
			want: true,
		},
		{
			name: "cancel failure",
			stub: &requestServiceStub{cancelErr: wantErr},
			invoke: func(h *handler.GenerationHandler) (bool, error) {
				response, err := h.Cancel(newHandlerContext(), dto.CancelGenerationRequest{GenerationRunID: 7})
				return response.Cancelled, err
			},
			wantErr: wantErr,
		},
		{
			name: "confirmation success",
			stub: &requestServiceStub{},
			invoke: func(h *handler.GenerationHandler) (bool, error) {
				response, err := h.ConfirmCandidate(newHandlerContext(), dto.ConfirmCandidateRequest{
					GenerationRunID: 7,
					CandidateID:     9,
				})
				return response.Confirmed, err
			},
			want: true,
		},
		{
			name: "confirmation failure",
			stub: &requestServiceStub{confirmErr: wantErr},
			invoke: func(h *handler.GenerationHandler) (bool, error) {
				response, err := h.ConfirmCandidate(newHandlerContext(), dto.ConfirmCandidateRequest{
					GenerationRunID: 7,
					CandidateID:     9,
				})
				return response.Confirmed, err
			},
			wantErr: wantErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.invoke(handler.NewGenerationHandler(test.stub))
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
			if got != test.want {
				t.Fatalf("expected result %t, got %t", test.want, got)
			}
		})
	}
}

func newHandlerContext() *echox.Context {
	e := echo.New()
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	return &echox.Context{Context: e.NewContext(request, httptest.NewRecorder())}
}
