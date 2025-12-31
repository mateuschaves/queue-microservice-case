import { Injectable, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { Pool } from 'pg';

@Injectable()
export class DatabaseService {
  private readonly logger = new Logger(DatabaseService.name);
  private pool: Pool;

  constructor(private configService: ConfigService) {
    this.pool = new Pool({
      host: this.configService.get<string>('DB_HOST', 'localhost'),
      port: this.configService.get<number>('DB_PORT', 5432),
      user: this.configService.get<string>('DB_USER', 'postgres'),
      password: this.configService.get<string>('DB_PASSWORD', 'postgres'),
      database: this.configService.get<string>('DB_NAME', 'queue_case'),
    });
  }

  async createMessage(
    idempotencyId: string,
    correlationId: string,
    payload: any,
  ): Promise<void> {
    const query = `
      INSERT INTO messages (idempotency_id, correlation_id, status, payload, created_at, updated_at)
      VALUES ($1, $2, 'pending', $3, NOW(), NOW())
      ON CONFLICT (idempotency_id) DO UPDATE SET updated_at = NOW()
    `;

    await this.pool.query(query, [idempotencyId, correlationId, JSON.stringify(payload)]);
  }

  async getMessage(id: string): Promise<any> {
    const query = `
      SELECT idempotency_id, correlation_id, status, payload, created_at, updated_at
      FROM messages
      WHERE idempotency_id = $1
    `;

    const result = await this.pool.query(query, [id]);
    if (result.rows.length === 0) {
      return null;
    }

    const row = result.rows[0];
    return {
      idempotency_id: row.idempotency_id,
      correlation_id: row.correlation_id,
      status: row.status,
      payload: typeof row.payload === 'string' ? JSON.parse(row.payload) : row.payload,
      created_at: row.created_at,
      updated_at: row.updated_at,
    };
  }

  async getMessageHistory(id: string): Promise<any[]> {
    const query = `
      SELECT id, idempotency_id, correlation_id, status, service_name, event_id, error_message, created_at
      FROM message_history
      WHERE idempotency_id = $1
      ORDER BY created_at ASC
    `;

    const result = await this.pool.query(query, [id]);
    return result.rows.map((row) => ({
      id: row.id,
      idempotency_id: row.idempotency_id,
      correlation_id: row.correlation_id,
      status: row.status,
      service_name: row.service_name,
      event_id: row.event_id,
      error_message: row.error_message,
      created_at: row.created_at,
    }));
  }
}

