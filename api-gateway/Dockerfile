FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files from api-gateway directory
COPY api-gateway/package*.json ./
RUN npm install

# Copy api-gateway source code
COPY api-gateway/ ./
RUN npm run build

FROM node:20-alpine

WORKDIR /app

COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package*.json ./

EXPOSE 3000

CMD ["npm", "run", "start:prod"]

