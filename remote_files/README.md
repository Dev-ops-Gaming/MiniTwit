# Scaling using Docker Swarm
This file explains how the scaling is set up

### Managers and Workers
An "n manager swarm tolerates loss of (n-1)/2 managers". Therefore, I've created 3 managers. If we need to make more, we have to use this command:
ssh root@$SWARM_MANAGER_IP "docker node promote NODE_NAME_TO_PROMOTE"

I've created 5 workers.

### Replica and Global Services
A service can be replicated or global.

Prometheus is set to be global, as it is responsible for gathering and shipping data about errors and general logging, and thus should run on every node, to collect all data.

For all the replicated services, I've just put the number of replicas I felt were enough for now. We can add more later if we need it.

### Container Names
"services.deploy.replicas: can't set container_name and app as container name must be unique: invalid compose project"
A container must have a unique name. Services with a set 'container_name', which are replicated, will result in multiple containers having that same name. Thus, it's not possible to use 'container_name' with swarm. I haven't found a workaround for this, which allows us to keep the names we have used till now.

### Allocating Ports
Replicas will all try to run on the port specified in the docker-compose.yml, but this could not be done with our old ports.

Using 'docker stack deploy' could maybe have solved this, but it throws errors for keywords like 'extends' and 'include' which we use for filebeat in the logging branch. It also cannot be passed an .env file.
I tried to solve this by 'merging' all the files together using 'docker compose config', but the output has a ton of stuff which docker stack doesn't support (also described in the official documentation). Manually changing the merged file would be too much work.

The solution right now is not the best
    ports:
      - "8080-8082:8080"
      #- "8080:8080"

Above should solve port allocation problems.