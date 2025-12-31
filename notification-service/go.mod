module queue-microservice-case/notification-service

go 1.21

require (
	queue-microservice-case/shared/contracts v0.0.0
	queue-microservice-case/shared/logger v0.0.0
	queue-microservice-case/shared/messaging v0.0.0
)

replace queue-microservice-case/shared/contracts => ../shared/contracts
replace queue-microservice-case/shared/logger => ../shared/logger
replace queue-microservice-case/shared/messaging => ../shared/messaging

