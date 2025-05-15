import { defineConfig } from '@hey-api/openapi-ts';

export default defineConfig({
  input: 'clabernetes-openapi.json',
  output: {
    case: "preserve",
    format: "biome",
    lint: "biome",
    path: 'src/lib/clabernetes-client'
  },
  plugins: [
    {
      name: "@hey-api/client-fetch",
    },
  ]
});
