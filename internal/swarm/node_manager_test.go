package swarm

import (
	"testing"

	dockerswarm "github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/assert"
)

func TestNodeManagerMapNodeMapsFields(t *testing.T) {
	node := dockerswarm.Node{
		ID: " node-1 ",
		Description: dockerswarm.NodeDescription{
			Hostname: " manager-1 ",
			Engine: dockerswarm.EngineDescription{
				EngineVersion: " 28.3.0 ",
			},
		},
		Status: dockerswarm.NodeStatus{
			State: dockerswarm.NodeStateReady,
			Addr:  " 10.0.0.1 ",
		},
		Spec: dockerswarm.NodeSpec{
			Availability: dockerswarm.NodeAvailabilityActive,
		},
		ManagerStatus: &dockerswarm.ManagerStatus{
			Leader: true,
		},
	}

	mapped := (&NodeManager{}).mapNode(node)

	assert.Equal(t, " node-1 ", mapped.ID, "unexpected id")
	assert.Equal(t, " manager-1 ", mapped.Hostname, "unexpected hostname")
	assert.Equal(t, "ready", mapped.Status, "unexpected status")
	assert.Equal(t, "active", mapped.Availability, "unexpected availability")
	assert.Equal(t, NodeManagerStatusLeader, mapped.ManagerStatus, "unexpected managerStatus")
	assert.Equal(t, " 28.3.0 ", mapped.EngineVersion, "unexpected engine version")
	assert.Equal(t, " 10.0.0.1 ", mapped.Addr, "unexpected addr")
}

func TestNodeManagerMapNodeSetsWorkerManagerStatusForWorkers(t *testing.T) {
	node := dockerswarm.Node{
		ID: "node-2",
	}

	mapped := (&NodeManager{}).mapNode(node)

	assert.Equal(t, NodeManagerStatusWorker, mapped.ManagerStatus, "worker node must have worker managerStatus")
}
