package broker

// Using NATS is simpler than RabbitMQ for the projects requirements.
var (
	StreamName        = "MESSAGES"
	SubjectGlobalRoom = StreamName + "." + "room.global"
)
