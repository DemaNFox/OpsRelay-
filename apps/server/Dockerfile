FROM node:22-alpine AS build

WORKDIR /app
RUN corepack enable

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml tsconfig.base.json ./
COPY packages/protocol/package.json packages/protocol/package.json
COPY apps/server/package.json apps/server/package.json
COPY apps/admin-cli/package.json apps/admin-cli/package.json

RUN pnpm install --frozen-lockfile

COPY packages/protocol packages/protocol
COPY apps/server apps/server
COPY apps/admin-cli apps/admin-cli

RUN pnpm --filter @opsrelay/protocol build
RUN pnpm --filter @opsrelay/server build
RUN pnpm --filter @opsrelay/admin-cli build

FROM node:22-alpine AS runtime

WORKDIR /app
ENV NODE_ENV=production
ENV OPSRELAY_SERVER_HOST=0.0.0.0
ENV OPSRELAY_SERVER_PORT=3030
ENV OPSRELAY_SERVER_URL=http://127.0.0.1:3030

RUN corepack enable

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml tsconfig.base.json ./
COPY packages/protocol/package.json packages/protocol/package.json
COPY apps/server/package.json apps/server/package.json
COPY apps/admin-cli/package.json apps/admin-cli/package.json
RUN pnpm install --frozen-lockfile --prod

COPY --from=build /app/packages/protocol/dist packages/protocol/dist
COPY --from=build /app/apps/server/dist apps/server/dist
COPY --from=build /app/apps/admin-cli/dist apps/admin-cli/dist
COPY apps/server/migrations apps/server/migrations

EXPOSE 3030
CMD ["node", "apps/server/dist/main.js"]
