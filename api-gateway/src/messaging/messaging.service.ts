import { Injectable, Logger, OnModuleDestroy } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import * as KafkaJS from 'kafkajs';
import * as amqp from 'amqplib';

@Injectable()
export class MessagingService implements OnModuleDestroy {
  private readonly logger = new Logger(MessagingService.name);
  private kafkaProducer: KafkaJS.Producer | null = null;
  private kafkaClient: KafkaJS.Kafka | null = null;
  private rabbitMQChannel: amqp.Channel | null = null;
  private rabbitMQConnection: amqp.Connection | null = null;
  private brokerType: string;

  constructor(private configService: ConfigService) {
    this.brokerType = this.configService.get<string>('MESSAGE_BROKER', 'kafka');
    this.initializeBroker();
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
    
    this.kafkaClient = new KafkaJS.Kafka({
      clientId: 'api-gateway',
      brokers,
    });

    this.kafkaProducer = this.kafkaClient.producer();
    await this.kafkaProducer.connect();

    this.logger.log('Kafka producer initialized');
  }

  private async initializeRabbitMQ() {
    const url = this.configService.get<string>(
      'RABBITMQ_URL',
      'amqp://guest:guest@localhost:5672/',
    );

    this.rabbitMQConnection = await amqp.connect(url);
    this.rabbitMQChannel = await this.rabbitMQConnection.createChannel();

    this.logger.log('RabbitMQ channel initialized');
  }

  async publish(topic: string, event: any): Promise<void> {
    if (this.brokerType === 'kafka' || this.brokerType === '') {
      await this.publishToKafka(topic, event);
    } else {
      await this.publishToRabbitMQ(topic, event);
    }
  }

  private async publishToKafka(topic: string, event: any): Promise<void> {
    if (!this.kafkaProducer) {
      throw new Error('Kafka producer not initialized');
    }

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
  }

  private async publishToRabbitMQ(queue: string, event: any): Promise<void> {
    if (!this.rabbitMQChannel) {
      throw new Error('RabbitMQ channel not initialized');
    }

    await this.rabbitMQChannel.assertQueue(queue, { durable: true });

    await this.rabbitMQChannel.sendToQueue(
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
  }

  async onModuleDestroy() {
    if (this.kafkaProducer) {
      await this.kafkaProducer.disconnect();
    }
    if (this.rabbitMQChannel) {
      await this.rabbitMQChannel.close();
    }
    if (this.rabbitMQConnection) {
      await this.rabbitMQConnection.close();
    }
  }
}

