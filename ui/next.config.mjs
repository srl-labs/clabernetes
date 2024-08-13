/** @type {import('next').NextConfig} */
const nextConfig = {
    output: "standalone",
    webpack: (config) => {
        config.module = {
            ...config.module,
            // suppress warning caused within package 'web-worker' which we only run on server side so should be fine
            exprContextCritical: false
        };
        return config;
    }
};

export default nextConfig;
