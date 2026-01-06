import { Injectable, Logger, OnModuleDestroy, OnModuleInit } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import * as KafkaJS from 'kafkajs';
import * as amqp from 'amqplib';

@Injectable()
export class MessagingService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(MessagingService.name);
  private kafkaProducer: KafkaJS.Producer | null = null;
  private kafkaClient: KafkaJS.Kafka | null = null;
  private rabbitMQChannel: amqp.Channel | null = null;
  private rabbitMQConnection: amqp.Connection | null = null;
  private brokerType: string;
  private isInitialized = false;
  private initializationPromise: Promise<void> | null = null;

  constructor(private configService: ConfigService) {
    this.brokerType = this.configService.get<string>('MESSAGE_BROKER', 'kafka');
  }

  async onModuleInit() {
    // Initialize broker with retry logic
    await this.initializeBrokerWithRetry();
  }

  private async initializeBrokerWithRetry(maxRetries = 10, delayMs = 5000) {
    for (let attempt = 1; attempt <= maxRetries; attempt++) {
      try {
        this.logger.log(`Attempting to initialize message broker (attempt ${attempt}/${maxRetries})...`);
        await this.initializeBroker();
        this.isInitialized = true;
        this.logger.log('Message broker initialized successfully');
        return;
      } catch (error) {
        this.logger.error(
          `Failed to initialize message broker (attempt ${attempt}/${maxRetries}): ${error.message}`,
        );
        
        if (attempt === maxRetries) {
          this.logger.error('Max retries reached. Service will continue but messaging may not work.');
          // Don't throw - allow service to start even if broker is not available
          // The service will retry on each publish attempt
          return;
        }
        
        this.logger.log(`Retrying in ${delayMs}ms...`);
        await this.sleep(delayMs);
      }
    }
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  private async initializeBroker() {
    if (this.brokerType === 'kafka' || this.brokerType === '') {
      await this.initializeKafka();
    } else if (this.brokerType === 'rabbit' || this.brokerType === 'rabbitmq') {
      await this.initializeRabbitMQ();
    } else {
      throw new Error(`Unsupported message broker: ${this.brokerType}`);
    }
  }

  private async initializeKafka() {
    const brokers = this.configService.get<string>('KAFKA_BROKERS', 'localhost:9092').split(',');
    
    this.logger.log(`Connecting to Kafka brokers: ${brokers.join(', ')}`);
    
    this.kafkaClient = new KafkaJS.Kafka({
      clientId: 'api-gateway',
      brokers,
      retry: {
        initialRetryTime: 100,
        retries: 8,
        multiplier: 2,
        maxRetryTime: 30000,
      },
      connectionTimeout: 10000,
      requestTimeout: 30000,
    });

    this.kafkaProducer = this.kafkaClient.producer({
      allowAutoTopicCreation: true,
      retry: {
        initialRetryTime: 100,
        retries: 5,
        multiplier: 2,
        maxRetryTime: 30000,
      },
    });

    try {
      await this.kafkaProducer.connect();
      this.logger.log('Kafka producer connected successfully');
    } catch (error) {
      this.logger.error(`Failed to connect Kafka producer: ${error.message}`);
      throw error;
    }
  }

  private async initializeRabbitMQ() {
    const url = this.configService.get<string>(
      'RABBITMQ_URL',
      'amqp://guest:guest@localhost:5672/',
    );

    this.logger.log(`Connecting to RabbitMQ: ${url.replace(/:[^:]*@/, ':****@')}`);
    
    try {
      this.rabbitMQConnection = await amqp.connect(url);
      this.rabbitMQChannel = await this.rabbitMQConnection.createChannel();
      this.logger.log('RabbitMQ channel initialized successfully');
    } catch (error) {
      this.logger.error(`Failed to connect RabbitMQ: ${error.message}`);
      throw error;
    }
  }

  async publish(topic: string, event: any): Promise<void> {
    // Ensure broker is initialized before publishing
    if (!this.isInitialized && !this.initializationPromise) {
      this.initializationPromise = this.initializeBrokerWithRetry();
    }
    
    if (this.initializationPromise) {
      await this.initializationPromise;
    }

    if (this.brokerType === 'kafka' || this.brokerType === '') {
      await this.publishToKafka(topic, event);
    } else {
      await this.publishToRabbitMQ(topic, event);
    }
  }

  private async publishToKafka(topic: string, event: any): Promise<void> {
    if (!this.kafkaProducer) {
      // Try to reconnect
      try {
        await this.initializeKafka();
      } catch (error) {
        this.logger.error(`Failed to reconnect Kafka producer: ${error.message}`);
        throw new Error('Kafka producer not available and reconnection failed');
      }
    }

    try {
      await this.kafkaProducer.send({
        topic,
        messages: [
          {
            key: event.idempotency_id,
            value: JSON.stringify(event),
            headers: {
              correlation_id: event.correlation_id,
              idempotency_id: event.idempotency_id,
              event_type: event.event_type,
            },
          },
        ],
      });

      this.logger.log(
        JSON.stringify({
          level: 'INFO',
          service: 'api-gateway',
          correlation_id: event.correlation_id,
          idempotency_id: event.idempotency_id,
          message: `Published event to Kafka topic: ${topic}`,
          timestamp: new Date().toISOString(),
        }),
      );
    } catch (error) {
      this.logger.error(
        `Failed to publish to Kafka: ${error.message}. Event: ${event.event_id}`,
      );
      // Check if connection is lost and try to reconnect
      if (error.message.includes('ECONNREFUSED') || error.message.includes('Connection closed')) {
        this.logger.log('Attempting to reconnect Kafka producer...');
        this.kafkaProducer = null;
        try {
          await this.initializeKafka();
          // Retry once after reconnection
          await this.kafkaProducer!.send({
            topic,
            messages: [
              {
                key: event.idempotency_id,
                value: JSON.stringify(event),
                headers: {
                  correlation_id: event.correlation_id,
                  idempotency_id: event.idempotency_id,
                  event_type: event.event_type,
                },
              },
            ],
          });
          this.logger.log('Successfully republished after reconnection');
        } catch (retryError) {
          this.logger.error(`Retry after reconnection failed: ${retryError.message}`);
          throw retryError;
        }
      } else {
        throw error;
      }
    }
  }

  private async publishToRabbitMQ(queue: string, event: any): Promise<void> {
    if (!this.rabbitMQChannel || !this.rabbitMQConnection) {
      // Try to reconnect
      try {
        await this.initializeRabbitMQ();
      } catch (error) {
        this.logger.error(`Failed to reconnect RabbitMQ: ${error.message}`);
        throw new Error('RabbitMQ channel not available and reconnection failed');
      }
    }

    try {
      // Try to assert queue, if it fails with PRECONDITION_FAILED, delete and recreate
      try {
        await this.rabbitMQChannel!.assertQueue(queue, { 
          durable: true,
          arguments: {
            'x-dead-letter-exchange': 'dlx'
          }
        });
      } catch (assertError: any) {
        if (assertError.message && assertError.message.includes('PRECONDITION_FAILED')) {
          this.logger.warn(`Queue ${queue} exists with different config, deleting and recreating...`);
          try {
            await this.rabbitMQChannel!.deleteQueue(queue);
            await this.rabbitMQChannel!.assertQueue(queue, { 
              durable: true,
              arguments: {
                'x-dead-letter-exchange': 'dlx'
              }
            });
          } catch (deleteError) {
            // If delete fails, try without DLX (queue might be in use)
            this.logger.warn(`Could not delete queue, trying to use existing queue...`);
            await this.rabbitMQChannel!.checkQueue(queue);
          }
        } else {
          throw assertError;
        }
      }

      await this.rabbitMQChannel!.sendToQueue(
        queue,
        Buffer.from(JSON.stringify(event)),
        {
          persistent: true,
          messageId: event.event_id,
          correlationId: event.correlation_id,
          headers: {
            correlation_id: event.correlation_id,
            idempotency_id: event.idempotency_id,
            event_type: event.event_type,
          },
        },
      );

      this.logger.log(
        JSON.stringify({
          level: 'INFO',
          service: 'api-gateway',
          correlation_id: event.correlation_id,
          idempotency_id: event.idempotency_id,
          message: `Published event to RabbitMQ queue: ${queue}`,
          timestamp: new Date().toISOString(),
        }),
      );
    } catch (error) {
      this.logger.error(
        `Failed to publish to RabbitMQ: ${error.message}. Event: ${event.event_id}`,
      );
      // Try to reconnect and retry
      if (error.message.includes('Connection closed') || error.message.includes('ECONNREFUSED')) {
        this.logger.log('Attempting to reconnect RabbitMQ...');
        this.rabbitMQChannel = null;
        this.rabbitMQConnection = null;
        try {
          await this.initializeRabbitMQ();
          await this.rabbitMQChannel!.assertQueue(queue, { durable: true });
          await this.rabbitMQChannel!.sendToQueue(
            queue,
            Buffer.from(JSON.stringify(event)),
            {
              persistent: true,
              messageId: event.event_id,
              correlationId: event.correlation_id,
              headers: {
                correlation_id: event.correlation_id,
                idempotency_id: event.idempotency_id,
                event_type: event.event_type,
              },
            },
          );
          this.logger.log('Successfully republished after reconnection');
        } catch (retryError) {
          this.logger.error(`Retry after reconnection failed: ${retryError.message}`);
          throw retryError;
        }
      } else {
        throw error;
      }
    }
  }

  async onModuleDestroy() {
    try {
      if (this.kafkaProducer) {
        await this.kafkaProducer.disconnect();
        this.logger.log('Kafka producer disconnected');
      }
    } catch (error) {
      this.logger.error(`Error disconnecting Kafka producer: ${error.message}`);
    }

    try {
      if (this.rabbitMQChannel) {
        await this.rabbitMQChannel.close();
        this.logger.log('RabbitMQ channel closed');
      }
    } catch (error) {
      this.logger.error(`Error closing RabbitMQ channel: ${error.message}`);
    }

    try {
      if (this.rabbitMQConnection) {
        await this.rabbitMQConnection.close();
        this.logger.log('RabbitMQ connection closed');
      }
    } catch (error) {
      this.logger.error(`Error closing RabbitMQ connection: ${error.message}`);
    }
  }
}

