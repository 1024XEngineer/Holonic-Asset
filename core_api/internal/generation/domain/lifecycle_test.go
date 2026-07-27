package domain_test

import (
	"reflect"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/generation/domain"
)

func TestRunLifecycleContract(t *testing.T) {
	got := []domain.RunLifecycle{
		domain.RunLifecycleAccepted,
		domain.RunLifecyclePlanning,
		domain.RunLifecyclePlanned,
		domain.RunLifecycleGenerating,
		domain.RunLifecyclePostProcessing,
		domain.RunLifecycleWaitingConfirmation,
		domain.RunLifecycleCompleted,
		domain.RunLifecycleFailed,
		domain.RunLifecycleCancelled,
	}
	want := []domain.RunLifecycle{
		"accepted",
		"planning",
		"planned",
		"generating",
		"post_processing",
		"waiting_confirmation",
		"completed",
		"failed",
		"cancelled",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected run lifecycle contract: %v", got)
	}
}
