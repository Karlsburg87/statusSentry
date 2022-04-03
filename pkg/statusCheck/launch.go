package statuscheck

import (
	"context"

	"github.com/karlsburg87/statusSentry/pkg/configuration"
	"github.com/karlsburg87/statusSentry/pkg/dispatch"
)

//Launch quickly launches status check operations - takes a context.Context with cancel
func Launch(ctx context.Context, configChan <-chan *configuration.Configuration) {
	directory := directory{
		configChan:  configChan,
		cancel:      ctx.Done(),
		rssChan:     make(chan configuration.Config),
		twitterChan: make(chan configuration.Config),
		sender:      make(chan configuration.Transporter),
		validators: validators{
			webhook: make(chan validator),
			email:   make(chan validator),
		},
	}

	go operator(directory)                                                 //the orchestrator goroutine - its pushing of a Config to a Pull type run function initiates the pull
	go dispatch.Sender("", directory.sender)                               //the goroutine handling sending messages to other microservices TODO: add microservice URL
	go runStatusWebhookServer(ctx, directory.validators, directory.sender) //also acts as server for email incoming updates
	go runRSSOperations(directory.rssChan, directory.sender)               //pulls RSS updates periodically
	go runTwitterOperations(directory.twitterChan, directory.sender)       //pulls Twitter updates periodically
}

//directory is a wrapper around all the goroutines handled by operator and spun up at Launch
type directory struct {
	configChan  <-chan *configuration.Configuration
	cancel      <-chan struct{}
	rssChan     chan configuration.Config
	twitterChan chan configuration.Config
	sender      chan configuration.Transporter //sender sends outgoing Transporters to a single http.Client for conn keep-alive efficiencies
	validators  validators
}

//validators houses the channels used by run functions to send fragments of info to operator function and
// receive a matching Config in return via validator.valid if a match exists.
//
//An empty Config object is returned through the validator.valid channel if no matches are found
type validators struct {
	webhook chan validator
	email   chan validator
}

//closePush closes the channels in the directory that are written to by the operator.
// Other channels must be closed by their writing functions
func (dir directory) closePush() {
	close(dir.rssChan)
	dir.rssChan = nil

	close(dir.twitterChan)
	dir.twitterChan = nil
}
