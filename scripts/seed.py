from sentence_transformers import SentenceTransformer

# 1. Download/Load the exact model we are using in Go
print("Loading model...")
model = SentenceTransformer('all-MiniLM-L6-v2')

# 2. Your two database resources
resource_1 = "Distributed Systems in Go Scalable backends"
resource_2 = "Sustainable Energy Renewable tech"

# 3. Generate the math
print("Calculating vectors...")
vec1 = model.encode(resource_1)
vec2 = model.encode(resource_2)

# 4. Format them perfectly for Postgres copy-pasting
print("\n--- COPY THIS FOR ID '1' (Go Systems) ---")
print("[" + ",".join([str(x) for x in vec1]) + "]")

print("\n--- COPY THIS FOR ID '2' (Energy) ---")
print("[" + ",".join([str(x) for x in vec2]) + "]")