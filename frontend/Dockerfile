# ---- dev stage ----
FROM node:20-alpine AS dev

WORKDIR /app

COPY package*.json ./
RUN npm install

# source is mounted at runtime via volume for hot reload
EXPOSE 5173
CMD ["npm", "run", "dev"]


# ---- prod stage ----
FROM node:20-alpine AS prod

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .
RUN npx vite build

EXPOSE 5173
CMD ["npx", "vite", "preview", "--host", "--port", "5173"]
