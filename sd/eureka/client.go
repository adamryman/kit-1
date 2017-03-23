package eureka

import (
	stdeureka "github.com/hudl/fargo"
	stdeurekalogging "github.com/op/go-logging"
)

func init() {
	// Quieten Fargo's own logging
	stdeurekalogging.SetLevel(stdeurekalogging.ERROR, "fargo")
}

// Client is a wrapper around the Eureka API.
type Client interface {
	// Register an instance with Eureka.
	Register(i *stdeureka.Instance) error

	// Deregister an instance from Eureka.
	Deregister(i *stdeureka.Instance) error

	// Send an instance heartbeat to Eureka.
	Heartbeat(i *stdeureka.Instance) error

	// Get all instances for an app in Eureka.
	Instances(app string) ([]*stdeureka.Instance, error)

	// Receive scheduled updates about an app's instances in Eureka.
	ScheduleUpdates(app string, quitc chan struct{}) <-chan stdeureka.AppUpdate
}

type client struct {
	connection *stdeureka.EurekaConnection
}

// NewClient returns an implementation of the Client interface, wrapping a
// concrete connection to Eureka using the Fargo library.
// Taking in Fargo's own connection abstraction gives the user maximum
// freedom in regards to how that connection is configured.
func NewClient(ec *stdeureka.EurekaConnection) Client {
	return &client{connection: ec}
}

func (c *client) Register(i *stdeureka.Instance) error {
	if c.instanceRegistered(i) {
		// Already registered. Send a heartbeat instead.
		return c.Heartbeat(i)
	}
	return c.connection.RegisterInstance(i)
}

func (c *client) Deregister(i *stdeureka.Instance) error {
	return c.connection.DeregisterInstance(i)
}

func (c *client) Heartbeat(i *stdeureka.Instance) (err error) {
	if err = c.connection.HeartBeatInstance(i); err != nil && c.instanceNotFoundErr(err) {
		// Instance not registered. Register first before sending heartbeats.
		return c.Register(i)
	}
	return err
}

func (c *client) Instances(app string) ([]*stdeureka.Instance, error) {
	stdApp, err := c.connection.GetApp(app)
	if err != nil {
		return nil, err
	}
	return stdApp.Instances, nil
}

func (c *client) ScheduleUpdates(app string, quitc chan struct{}) <-chan stdeureka.AppUpdate {
	return c.connection.ScheduleAppUpdates(app, false, quitc)
}

func (c *client) instanceRegistered(i *stdeureka.Instance) bool {
	_, err := c.connection.GetInstance(i.App, i.Id())
	return err == nil
}

func (c *client) instanceNotFoundErr(err error) bool {
	code, ok := stdeureka.HTTPResponseStatusCode(err)
	return ok && code == 404
}
