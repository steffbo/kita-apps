import createClient from 'openapi-fetch';
import type { paths } from './schema';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';

export const apiClient = createClient<paths>({
  baseUrl: API_BASE_URL,
});

// Add auth header interceptor
export function setAuthToken(token: string | null) {
  if (token) {
    apiClient.use({
      onRequest: ({ request }) => {
        request.headers.set('Authorization', `Bearer ${token}`);
        return request;
      },
    });
  }
}

export * from './schema';
