import { KubeConfig } from "@kubernetes/client-node";
import { Buffer } from "node:buffer";
import fs from "node:fs";

function decode(str: string): string {
  return Buffer.from(str, "base64").toString("binary");
}

const inClusterTokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token";
const inClusterRootCaFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt";

interface ClientOpts {
  host: string;
  token: string;
  ca: string;
  caDecoded: string;
  clientCert: string;
  clientCertDecoded: string;
  key: string;
  keyDecoded: string;
}

function getInClusterClientOpts(): ClientOpts {
  const host = process.env.KUBERNETES_SERVICE_HOST;
  const port = process.env.KUBERNETES_SERVICE_PORT;
  const token = fs.readFileSync(inClusterTokenFile, "utf8");
  const ca = fs.readFileSync(inClusterRootCaFile, "utf8");

  return {
    ca: ca,
    caDecoded: "",
    clientCert: "",
    clientCertDecoded: "",
    host: `https://${host}:${port}`,
    key: "",
    keyDecoded: "",
    token: token,
  };
}

function getClientOpts(): ClientOpts {
  const kc = new KubeConfig();
  kc.loadFromDefault();

  const opts = {
    ca: "",
    caDecoded: "",
    clientCert: "",
    clientCertDecoded: "",
    host: "",
    key: "",
    keyDecoded: "",
    token: "",
  };

  const cluster = kc.getCurrentCluster();

  opts.ca = cluster?.caData ?? opts.ca;

  opts.host = cluster?.server ?? opts.host;

  const user = kc.getCurrentUser();

  opts.clientCert = user?.certData ?? opts.clientCert;

  opts.key = user?.keyData ?? opts.key;

  opts.token = user?.token ?? opts.token;

  return opts;
}

function loadOptions(): ClientOpts {
  let opts = {
    ca: "",
    caDecoded: "",
    clientCert: "",
    clientCertDecoded: "",
    host: "",
    key: "",
    keyDecoded: "",
    token: "",
  };

  try {
    opts = getInClusterClientOpts();
  } catch (_error) {
    // oh no... anyway
  }

  if (opts.host !== "" || opts.token !== "" || opts.ca !== "") {
    // service account ca stuff is already decoded
    opts.caDecoded = opts.ca;

    return opts;
  }

  opts = getClientOpts();

  if (opts.host === "") {
    throw new Error("failed fetching kubeconfig options!");
  }

  if (opts.ca !== "") {
    opts.caDecoded = decode(opts.ca);
  }

  if (opts.ca !== "") {
    opts.clientCertDecoded = decode(opts.clientCert);
  }

  if (opts.key !== "") {
    opts.keyDecoded = decode(opts.key);
  }

  return opts;
}

export const kubeConfigOptions = loadOptions();
