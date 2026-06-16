import { createApp } from "./app.js";
import { loadConfig } from "./config.js";

const config = loadConfig(process.env);
const { app } = await createApp({ config });

await app.listen({ host: config.host, port: config.port });
