# Basic configuration example

This example contains basic configurations for deploy public repositories.
- `pull` mode
- non-auth polling
- init jobs (for secrets rotation)
- UI and API on `8080` port (`GET /api/v1/stacks`, `POST /api/v1/sync`)
- Health Server on `8082` port

Your steps:
- Add secret `printf 'change-me' | docker secret create db_password -`
- Add config `docker config create api_env ./api_env`
- Run `docker stack deploy --with-registry-auth -c docker-compose.yaml swarm-deploy --detach=false`
