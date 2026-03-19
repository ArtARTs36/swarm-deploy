package webserver

import (
	"github.com/artarts36/swarm-deploy/internal/controller"
	generated "github.com/artarts36/swarm-deploy/internal/entrypoints/webserver/generated"
)

func toGeneratedStacks(stacks []controller.StackView) []generated.StackView {
	mapped := make([]generated.StackView, 0, len(stacks))

	for _, stack := range stacks {
		mapped = append(mapped, toGeneratedStack(stack))
	}

	return mapped
}

func toGeneratedStack(stack controller.StackView) generated.StackView {
	mapped := generated.StackView{
		Name:        stack.Name,
		ComposeFile: stack.ComposeFile,
		LastStatus:  stack.LastStatus,
		Services:    toGeneratedServices(stack.Services),
	}

	if stack.LastError != "" {
		mapped.LastError = generated.NewOptString(stack.LastError)
	}
	if stack.LastCommit != "" {
		mapped.LastCommit = generated.NewOptString(stack.LastCommit)
	}
	if !stack.LastDeployAt.IsZero() {
		mapped.LastDeployAt = generated.NewOptDateTime(stack.LastDeployAt)
	}
	if stack.SourceDigest != "" {
		mapped.SourceDigest = generated.NewOptString(stack.SourceDigest)
	}

	return mapped
}

func toGeneratedServices(services []controller.ServiceView) []generated.ServiceView {
	mapped := make([]generated.ServiceView, 0, len(services))

	for _, service := range services {
		mappedService := generated.ServiceView{
			Name: service.Name,
		}
		if service.Image != "" {
			mappedService.Image = generated.NewOptString(service.Image)
		}
		if service.ImageVersion != "" {
			mappedService.ImageVersion = generated.NewOptString(service.ImageVersion)
		}
		if service.LastStatus != "" {
			mappedService.LastStatus = generated.NewOptString(service.LastStatus)
		}
		if !service.LastDeployAt.IsZero() {
			mappedService.LastDeployAt = generated.NewOptDateTime(service.LastDeployAt)
		}

		mapped = append(mapped, mappedService)
	}

	return mapped
}
