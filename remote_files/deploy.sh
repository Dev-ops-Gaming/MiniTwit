#!/bin/bash

source ~/.bash_profile

cd /minitwit

echo "Pulling latest images..."
docker-compose pull

# Deploy to Docker Swarm using stack deploy
echo "Deploying stack to Docker Swarm..."
docker stack deploy -c docker-compose.yml minitwit

echo "Current running services in Swarm:"
docker service ls
