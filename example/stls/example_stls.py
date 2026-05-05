from scripts.wrapper import stls

# 1. Melakukan PUT Request (Update data)
response_put = stls.put(
    url="https://api.target.com/v1/users/123",
    headers={"Authorization": "Bearer token"},
    data='{"role": "admin"}',
)

# 2. Melakukan DELETE Request
response_delete = stls.delete(
    url="https://api.target.com/v1/users/123", headers={"Authorization": "Bearer token"}
)

# 3. Melakukan PATCH Request
response_patch = stls.patch(
    url="https://api.target.com/v1/users/123", data='{"status": "active"}'
)
