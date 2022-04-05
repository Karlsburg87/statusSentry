#!/usr/bin/env bash
#Process mangement: https://www.digitalocean.com/community/tutorials/how-to-use-bash-s-job-control-to-manage-foreground-and-background-processes

#Find if programs are installed using command built-in function
#See: http://manpages.ubuntu.com/manpages/trusty/man1/bash.1.html (search: 'Run  command  with  args')
if [[ -x "$(command -v podman)" ]]; then
  
    podman build -f Dockerfile -t statussentry . \
    --build-arg PROJECT_ID=statusSentry \
    --build-arg CONFIG_LOCATION="https://configlocation.com" \
    --build-arg TWITTER_TOKEN=XXX \
    --build-arg OUTBOUND_URL="https://someurl.com" \
    --build-arg GOOGLE_APPLICATION_CREDENTIALS=XXX \
    --build-arg STATUS_UPDATE_TOPIC=statusUpdate \
    --build-arg PING_RESPONSE_TOPIC=pinger \
    --build-arg STATUS_CHECK_ONLY=false \
    --build-arg PINGER_ONLY=false \
    --build-arg PORT=8080 
    
    podman run --rm --name sentry -t -p 8080:8080 -p 8099:8099 statussentry

    #cleanup
    #podman image prune

elif [[ -x "$(command -v docker)" ]]; then
  
    docker build -f Dockerfile -t statussentry . \
    --build-arg PROJECT_ID=statusSentry \
    --build-arg CONFIG_LOCATION="https://configlocation.com" \
    --build-arg TWITTER_TOKEN=XXX \
    --build-arg OUTBOUND_URL="https://someurl.com" \
    --build-arg GOOGLE_APPLICATION_CREDENTIALS=XXX \
    --build-arg STATUS_UPDATE_TOPIC=statusUpdate \
    --build-arg PING_RESPONSE_TOPIC=pinger \
    --build-arg STATUS_CHECK_ONLY=false \
    --build-arg PINGER_ONLY=false \
    --build-arg PORT=8080 
    docker run --rm --name sentry -dt -p 8080:8080 -p 8099:8099 statussentry

    #cleanup
    docker image prune
else
  echo "You need to have either Docker Desktop or Podman to run"
fi