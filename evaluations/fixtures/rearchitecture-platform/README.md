# Shared platform clients

Orders and payments each construct their own HTTP client and retry policy.
The platform team wants one cross-cutting client without changing service-level
timeouts or making a provider switch based on preference alone.
