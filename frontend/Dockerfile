# Build stage
FROM node:18-alpine as build

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install

# Copy source code and public files
COPY . .

# Build the app
RUN npm run build

# Production stage
FROM node:18-alpine

WORKDIR /app

# Install serve
RUN npm install -g serve

# Copy built assets from build stage
COPY --from=build /app/build ./build

# Create directory for dynamic files
RUN mkdir -p /app/build/dynamic

# Expose port 3000
EXPOSE 3000

# Start serve
CMD ["serve", "-s", "build", "-l", "3000"] 