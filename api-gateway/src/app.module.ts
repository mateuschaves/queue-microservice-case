import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { MessagesController } from './messages/messages.controller';
import { MessagesService } from './messages/messages.service';
import { MessagingModule } from './messaging/messaging.module';
import { DatabaseModule } from './database/database.module';

@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
    }),
    MessagingModule,
    DatabaseModule,
  ],
  controllers: [MessagesController],
  providers: [MessagesService],
})
export class AppModule {}

