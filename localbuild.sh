#!/usr/bin/env bash
#Process mangement: https://www.digitalocean.com/community/tutorials/how-to-use-bash-s-job-control-to-manage-foreground-and-background-processes

#Find if programs are installed using command built-in function
#See: http://manpages.ubuntu.com/manpages/trusty/man1/bash.1.html (search: 'Run  command  with  args')
if [[ -x "$(command -v podman)" ]]; then
  
    podman build -f Dockerfile -t statussentry \
    --build-arg PROJECT_ID="${PROJECT_ID}" \
    --build-arg CONFIG_LOCATION="${CONFIG_LOCATION}" \
    --build-arg TWITTER_TOKEN="${TWITTER_TOKEN}" \
    --build-arg OUTBOUND_URL="${OUTBOUND_URL}" \
    --build-arg GOOGLE_APPLICATION_CREDENTIALS="${GOOGLE_APPLICATION_CREDENTIALS}" \
    --build-arg STATUS_UPDATE_TOPIC="${STATUS_UPDATE_TOPIC}" \
    --build-arg PING_RESPONSE_TOPIC="${PING_RESPONSE_TOPIC}" \
    --build-arg STATUS_CHECK_ONLY="${STATUS_CHECK_ONLY}" \
    --build-arg PINGER_ONLY="${PINGER_ONLY}" \
    --build-arg PORT="${PORT}" \
    .
    
    podman run --rm --name sentry -t -p 8080:8080 -p 8099:8099 statussentry

    #cleanup
    #podman image prune

elif [[ -x "$(command -v docker)" ]]; then
  
    docker build -f Dockerfile -t statussentry \
    --build-arg PROJECT_ID="${PROJECT_ID}" \
    --build-arg CONFIG_LOCATION="${CONFIG_LOCATION}" \
    --build-arg TWITTER_TOKEN="${TWITTER_TOKEN}" \
    --build-arg OUTBOUND_URL="${OUTBOUND_URL}" \
    --build-arg GOOGLE_APPLICATION_CREDENTIALS="${GOOGLE_APPLICATION_CREDENTIALS}" \
    --build-arg STATUS_UPDATE_TOPIC="${STATUS_UPDATE_TOPIC}" \
    --build-arg PING_RESPONSE_TOPIC="${PING_RESPONSE_TOPIC}" \
    --build-arg STATUS_CHECK_ONLY="${STATUS_CHECK_ONLY}" \
    --build-arg PINGER_ONLY="${PINGER_ONLY}" \
    --build-arg PORT="${PORT}" \
    .

    docker run --rm --name sentry -dt -p 8080:8080 -p 8099:8099 statussentry

    #cleanup
    docker image prune
else
  echo "You need to have either Docker Desktop or Podman to run"
fi