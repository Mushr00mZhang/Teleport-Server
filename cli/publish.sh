docker load <teleport_server.tar
docker compose -p telteport_server down
docker compose -p telteport_server up -d --remove-orphans
