FROM oven/bun:1.1.29

ENV DEBIAN_FRONTEND=noninteractive

WORKDIR /app

# Copy package files
COPY package.json bun.lockb* ./

# Install dependencies
RUN bun install

# Install Playwright browsers and all required system dependencies
RUN bunx playwright install --with-deps chromium

# Copy app files
COPY sync.js upload.js server.js ./
COPY entrypoint.sh ./

# Make entrypoint executable
RUN chmod +x entrypoint.sh

# Environment defaults
ENV NODE_ENV=production \
    HEADLESS=true

# Run sync once then exit
CMD ["./entrypoint.sh"]
