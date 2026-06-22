FROM node:26-alpine AS deps
WORKDIR /app
COPY package.json ./
RUN npm install

FROM node:26-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:26-alpine
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3456
ENV INTERNAL_API_BASE_URL=http://backend:8671
ENV MODEX_PUBLIC_API_BASE_URL=http://localhost:8671
ENV MODEX_PUBLIC_KROKI_URL=https://kroki.io
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
COPY --from=build /app/public ./public
COPY docker-entrypoint.sh /usr/local/bin/modex-frontend-entrypoint
RUN chmod +x /usr/local/bin/modex-frontend-entrypoint
EXPOSE 3456
ENTRYPOINT ["modex-frontend-entrypoint"]
CMD ["node", "server.js"]
