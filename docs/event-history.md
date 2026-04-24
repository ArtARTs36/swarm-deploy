## Event history

| Type                               | Severity | Category   | Trigger                             | Details keys                                                 |
|------------------------------------|----------|------------|-------------------------------------|--------------------------------------------------------------|
| `deploySuccess`                    | `info`   | `sync`     | Successful stack deployment         | `stack`, `commit`                                            |
| `deployFailed`                     | `alert`  | `sync`     | Failed stack deployment             | `stack`, `commit`, `error` (if present)                      |
| `sendNotificationFailed`           | `error`  | `sync`     | Notification delivery failure       | `destination`, `channel`, `event_type`, `error` (if present) |
| `syncManualStarted`                | `info`   | `sync`     | Manual sync run started             | `triggered_by` (if present)                                  |
| `serviceMissed`                    | `alert`  | `sync`     | Service disappeared from cluster    | `stack_name`, `service_name`                                 |
| `serviceRestored`                  | `info`   | `sync`     | Service restored after drift        | `stack_name`, `service_name`                                 |
| `serviceRestoreFailed`             | `alert`  | `sync`     | Service restore failed after drift  | `stack_name`, `service_name`                                 |
| `serviceReplicasDiverged`          | `warn`   | `sync`     | Runtime replicas diverged from spec | `stack_name`, `service_name`                                 |
| `serviceReplicasIncreased`         | `info`   | `sync`     | Service replicas count increased    | `stack`, `service`, `previous_replicas`, `current_replicas`, `username` (if present) |
| `serviceReplicasDecreased`         | `info`   | `sync`     | Service replicas count decreased    | `stack`, `service`, `previous_replicas`, `current_replicas`, `username` (if present) |
| `serviceRestarted`                 | `info`   | `sync`     | Service restarted                   | `stack`, `service`, `username` (if present)                  |
| `userAuthenticated`                | `info`   | `security` | User passed web authentication      | `username`                                                   |
| `assistantPromptInjectionDetected` | `alert`  | `security` | Assistant prompt injection detected | `detector`, `prompt` (if present), `username` (if present)   |

All runtime events are persisted to disk in `.swarm-deploy/event-history.json` and can be viewed via API:

- `GET /api/v1/events` - returns latest stored events
  - optional query filters:
    - `severities` - list of severities (`info`, `warn`, `error`, `alert`)
    - `categories` - list of categories (`sync`, `security`)

History size is bounded by `eventHistory.capacity` in config. When limit is reached, the oldest event is removed.

Config example:
```yaml
# Event history configuration.
eventHistory:
  capacity: 500
```
