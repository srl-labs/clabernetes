import { kubeConfigOptions } from "@/lib/kubeconfig.ts";
import { createClient } from "@hey-api/client-fetch";
import { Agent, setGlobalDispatcher } from "undici";
import { client } from "@/lib/clabernetes-client";

const agent = new Agent({
  connect: {
    ca: kubeConfigOptions.caDecoded,
    cert: kubeConfigOptions.clientCertDecoded,
    keepAlive: false,
    key: kubeConfigOptions.keyDecoded,
  },
});

setGlobalDispatcher(agent);

createClient({
  baseUrl: kubeConfigOptions.host,
  cache: "no-store",
  headers: {
    // add the header -- for local/dev mode i guess we wouldnt need this anyway but it doesnt seem
    // to hurt so just leave it in always
    authorization: `Bearer ${kubeConfigOptions.token}`,
  },
});

// TODO do we need *both* like this one is for the "internal" client in the generated stuff, but
//  what about for the normal kube client things?
client.setConfig({
  baseUrl: kubeConfigOptions.host,
  cache: "no-store",
  headers: {
    // add the header -- for local/dev mode i guess we wouldnt need this anyway but it doesnt seem
    // to hurt so just leave it in always
    authorization: `Bearer ${kubeConfigOptions.token}`,
  },
});
