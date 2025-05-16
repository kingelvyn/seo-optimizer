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

# Ensure favicon and other assets are properly copied
RUN cp -f public/favicon.ico build/ && \
    cp -f public/robots.txt build/ && \
    cp -f public/sitemap.xml build/ && \
    cp -f public/sitemap.xsl build/

# Production stage
FROM node:18-alpine

WORKDIR /app

# Install serve
RUN npm install -g serve

# Copy built assets from build stage
COPY --from=build /app/build ./build

# Add cache control headers for favicon
RUN echo '{"headers": [{"source": "favicon.ico", "headers": [{"key": "Cache-Control", "value": "no-cache, no-store, must-revalidate"}]}]}' > build/serve.json

# Expose port 3000
EXPOSE 3000

# Start serve with custom config
CMD ["serve", "-s", "build", "-l", "3000"] 