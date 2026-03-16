import { createClient } from "../../api/client.gen.js";

type ByodbClient = {
  users: {
    UserRegister: (request: {
      email: string;
      name: string;
      password: string;
    }) => Promise<unknown>;
    UserConfirm: (request: { code: string }) => Promise<unknown>;
    UserLogin: (request: { email: string; password: string }) => Promise<unknown>;
    UserMe: (options?: { auth?: string }) => Promise<unknown>;
  };
  admin: {
    UsersGetMany: (options?: { auth?: string }) => Promise<unknown>;
    UserByID: (request: { id: string }, options?: { auth?: string }) => Promise<unknown>;
    UserDisable: (request: { id: string }, options?: { auth?: string }) => Promise<unknown>;
    UserCreate: (
      request: { email: string; name: string; role: string; password?: string },
      options?: { auth?: string },
    ) => Promise<unknown>;
  };
  states: {
    StatesGetMany: () => Promise<unknown>;
    StateByCode: (request: { code: string }) => Promise<unknown>;
    StateCreate: (request: { code: string; name: string }) => Promise<unknown>;
  };
};

export const api = createClient(window.location.origin) as ByodbClient;
