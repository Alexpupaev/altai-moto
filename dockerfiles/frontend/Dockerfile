FROM node:20-alpine

WORKDIR /app

# Зависимости — отдельный слой, пересобирается только при изменении package.json
COPY frontend/package.json frontend/package-lock.json ./
RUN npm install

EXPOSE 3000

CMD ["npm", "run", "dev"]
