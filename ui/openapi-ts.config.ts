import { defineConfig } from '@hey-api/openapi-ts';

export default defineConfig({
  client: '@hey-api/client-fetch',
  input: 'clabernetes-openapi.json',
  output: {
    format: "biome",
    lint: "biome",
    path: 'src/lib/clabernetes-client'
  },
  types: {
    name: 'PascalCase',
  },
});