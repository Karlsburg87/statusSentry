FROM docker.io/library/golang:1-alpine AS build-env
WORKDIR /go/src/statusSentry

#Let us cache modules retrieval as they do not change often.
#Better use of cache than go get -d -u
COPY go.mod .
COPY go.sum .
RUN go mod download

#Update certificates
RUN apk --update add ca-certificates

#Get source and build binary
COPY . .

#Path to main function
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /statusSentry/bin

#Production image - scratch is the smallest possible but Alpine is a good second for bash-like access
FROM scratch
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /statusSentry/bin /bin/statusSentry

#Default root user container envars
ARG CONFIG_LOCATION
ARG PORT="8080"
ARG TWITTER_TOKEN
ARG OUTBOUND_URL
#GCP Specific
ARG GOOGLE_APPLICATION_CREDENTIALS
ARG STATUS_UPDATE_TOPIC
ARG PING_RESPONSE_TOPIC
ARG PROJECT_ID

ENV CONFIG_LOCATION=${CONFIG_LOCATION}
ENV PORT=${PORT}
ENV TWITTER_TOKEN=${TWITTER_TOKEN}
ENV OUTBOUND_URL=${OUTBOUND_URL}
#GCP Specific
ENV GOOGLE_APPLICATION_CREDENTIALS=${GOOGLE_APPLICATION_CREDENTIALS}
ENV STATUS_UPDATE_TOPIC=${STATUS_UPDATE_TOPIC}
ENV PING_RESPONSE_TOPIC=${PING_RESPONSE_TOPIC}
ENV PROJECT_ID=${PROJECT_ID}

#Expose port for webhook server
EXPOSE 8080
#Expose port for config update requests
EXPOSE 8099

CMD ["/bin/statusSentry"]