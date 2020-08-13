package subscriber

const (
	channelNameUserCreated = "userCreated"

	maxNewUsersAllowed = 100
)

//func (s *Subscriber) HandleNewUser(ctx context.Context, msg amqp.Delivery) error {
//// Decode incoming message
//var user proto.User
//if err := s.Encoder.Decode(msg.Body, &user); err != nil {
//	return errors.Wrap(err, "failed to decode new user")
//}
//
//// Handle incoming message somehow
//if err := s.Handler.CreateUser(context.Background(), &user); err != nil {
//	return errors.Wrap(err, "failed to create a user")
//}
//
//logs.Eloger.WithFields(logrus.Fields{
//	"event": "handling new user",
//}).Info("Bootstrap created new user successfully")
//return nil
//}

//func (s *Subscriber) initUserCreated() error {
//	userCreatedChannel, err := s.Rabbit.GetReceiver(channelNameUserCreated)
//	if err != nil {
//		return errors.Wrapf(err, "failed to get a receiver channel")
//	}
//
//	// Declare an exchange first
//	err = userCreatedChannel.ExchangeDeclare(
//		"int.bootstrap.debug", // name // TODO: use constant or variable
//		"direct",              // type // TODO: change me
//		true,                  // durable
//		false,                 // auto-deleted
//		false,                 // internal
//		false,                 // no-wait
//		nil,                   // arguments
//	)
//	if err != nil {
//		return errors.Wrap(err, "failed to create an exchange")
//	}
//
//	queue, err := userCreatedChannel.QueueDeclare(
//		"bootstrap.user.new", // name // TODO: use constant or variable
//		true,                 // durable
//		false,                // delete when unused
//		false,                // exclusive
//		false,                // no-wait
//		nil,                  // arguments
//	)
//	if err != nil {
//		return errors.Wrap(err, "failed to declare a queue")
//	}
//
//	// TODO: use some constants or variables
//	err = userCreatedChannel.QueueBind(
//		queue.Name,            // queue name
//		"new",                 // routing key
//		"int.bootstrap.debug", // exchange
//		false,
//		nil,
//	)
//	if err != nil {
//		return errors.Wrap(err, "failed to bind a queue")
//	}
//
//	msgs, err := userCreatedChannel.Consume(
//		queue.Name,                 // queue
//		"bootstrap.user.new.debug", // consumer // TODO: use some constant or variable
//		true,                       // auto-ack
//		false,                      // exclusive
//		false,                      // no-local
//		false,                      // no-wait
//		nil,                        // args
//	)
//	if err != nil {
//		return errors.Wrap(err, "failed to consume from a channel")
//	}
//
//	s.wg.Add(1)
//	go s.handleNewUsers(msgs) // handle incoming messages
//	return nil
//}
//
//func (s *Subscriber) handleNewUsers(messages <-chan amqp.Delivery) {
//	defer s.wg.Done()
//
//	limit := limiter.NewConcurrencyLimiter(maxNewUsersAllowed)
//	defer limit.Wait()
//
//	for {
//		select {
//		case <-s.closed:
//			return
//		case msg := <-messages:
//			// Start new goroutine to handle multiple requests at the same time
//			limit.Execute(lang.Recover(
//				func() {
//					if err := s.HandleNewUser(context.Background(), msg); err != nil {
//						logs.Eloger.Errorf("failed to handle new user: %v", err)
//					}
//				},
//			))
//		}
//	}
//}
