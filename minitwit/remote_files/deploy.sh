#!/bin/bash

source ~/.bash_profile

cd /minitwit

echo "Pulling latest images..."
docker-compose pull

echo "Restarting services..."
docker-compose up -d

echo "Current running containers:"
docker ps