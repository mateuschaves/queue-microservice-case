import { Injectable, Logger } from '@nestjs/common';
import { v4 as uuidv4 } from 'uuid';
import { CreateMessageDto } from './dto/create-message.dto';
import { MessagingService } from '../messaging/messaging.service';
import { DatabaseService } from '../database/database.service';

@Injectable()
export class MessagesService {
  private readonly logger = new Logger(MessagesService.name);
  private readonly serviceName = 'api-gateway';

  constructor(
    private readonly messagingService: MessagingService,
    private readonly databaseService: DatabaseService,
  ) {}

  async createMessage(createMessageDto: CreateMessageDto) {
    // Generate correlation_id and idempotency_id
    const correlationId = uuidv4();
    const idempotencyId = uuidv4();

    this.logger.log(
      JSON.stringify({
        level: 'INFO',
        service: this.serviceName,
        correlation_id: correlationId,
        idempotency_id: idempotencyId,
        message: 'Creating new message',
        timestamp: new Date().toISOString(),
      }),
    );

    // Create payload
    const payload = {
      content: createMessageDto.content,
      metadata: createMessageDto.metadata || {},
    };

    // Create event
    const event = {
      event_id: uuidv4(),
      correlation_id: correlationId,
      idempotency_id: idempotencyId,
      event_type: 'message.created',
      source_service: this.serviceName,
      timestamp: new Date().toISOString(),
      payload,
    };

    // Store in database
    await this.databaseService.createMessage(
      idempotencyId,
      correlationId,
      payload,
    );

    // Publish event
    await this.messagingService.publish('message.created', event);

    this.logger.log(
      JSON.stringify({
        level: 'INFO',
        service: this.serviceName,
        correlation_id: correlationId,
        idempotency_id: idempotencyId,
        message: 'Message created and event published',
        timestamp: new Date().toISOString(),
      }),
    );

    return {
      id: idempotencyId,
      correlation_id: correlationId,
      idempotency_id: idempotencyId,
      status: 'pending',
    };
  }

  async getMessageStatus(id: string) {
    const message = await this.databaseService.getMessage(id);
    
    if (!message) {
      return {
        id,
        status: 'not_found',
      };
    }

    const history = await this.databaseService.getMessageHistory(id);

    return {
      id: message.idempotency_id,
      correlation_id: message.correlation_id,
      status: message.status,
      created_at: message.created_at,
      updated_at: message.updated_at,
      history: history.map((h) => ({
        status: h.status,
        service: h.service_name,
        event_id: h.event_id,
        error: h.error_message,
        timestamp: h.created_at,
      })),
    };
  }
}

