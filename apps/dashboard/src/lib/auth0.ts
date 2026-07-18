import { Auth0Client } from "@auth0/nextjs-auth0/server";

// Reads AUTH0_DOMAIN, AUTH0_CLIENT_ID, AUTH0_CLIENT_SECRET, AUTH0_SECRET and
// APP_BASE_URL from the environment. The audience makes Auth0 issue a JWT
// access token for our Go API instead of an opaque one.
export const auth0 = new Auth0Client({
  authorizationParameters: {
    audience: process.env.AUTH0_AUDIENCE ?? "https://api.devopsaccess.in",
    scope: "openid profile email",
  },
});
