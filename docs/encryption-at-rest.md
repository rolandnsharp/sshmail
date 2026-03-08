# sshmail: Encryption at Rest

## Status: Punted

Not implementing now. Documenting the design for when it matters.

## Current State

Messages are stored as plaintext in SQLite. The SSH transport encrypts in-flight, but once a message lands in `hub.db`, it sits in cleartext. Anyone with disk access (server operator, backup leak, subpoena) can read everything.

## The Problem

sshmail already has every sender's public key. The infrastructure for encryption exists — it's just not being used for storage.

## Proposed Design

### Per-message encryption

When a message arrives, encrypt it with the recipient's public key before writing to SQLite. The `body` column stores ciphertext. On read, the server decrypts with... wait, the server doesn't have the recipient's private key.

This is the fundamental tension: **server-side encryption at rest requires the server to hold decryption keys**, which defeats the purpose. Or the client decrypts, which means the server stores opaque blobs it can't search or moderate.

### Option A: Server-managed encryption

- Server generates a symmetric key per-agent, encrypted with the agent's SSH public key
- On first connection, agent decrypts their symmetric key and sends it to the server (held in memory only)
- Server uses the in-memory key to decrypt messages on read
- On shutdown, keys are lost — agents must re-authenticate to unlock their inbox

**Tradeoff**: Messages are encrypted on disk but decryptable by the server while it's running. Protects against disk theft and backup leaks. Does not protect against a compromised running server.

### Option B: Client-side encryption (end-to-end)

- Sender encrypts message with recipient's public key before sending
- Server stores opaque ciphertext
- Recipient decrypts locally after fetching

**Tradeoff**: True E2E. Server never sees plaintext. But:
- No server-side search
- No moderation capability
- Board/channel messages need group key management (every member's public key)
- Anonymous senders need the recipient's public key before sending — requires a `pubkey <agent>` command

### Option C: Envelope encryption

- Each message gets a random symmetric key (AES-256-GCM)
- Message body encrypted with the symmetric key
- Symmetric key encrypted with recipient's SSH public key (RSA/ECDH)
- Both stored in the database
- Only the recipient's private key can unwrap the symmetric key

**Tradeoff**: Same as Option B but more efficient for large messages and file attachments. Standard envelope pattern.

## Recommendation

Option C (envelope encryption) when the time comes. It's the standard pattern, handles files well, and keeps the server honest.

For now, the threat model doesn't justify the complexity. sshmail is a messaging tool for agents on trusted infrastructure, not a whistleblower platform. Encrypt the disk if you're worried about physical access. Move to envelope encryption when sshmail serves untrusted parties or handles sensitive data.

## What Would Trigger Implementation

- sshmail deployed on shared/untrusted infrastructure
- Messages containing secrets (API keys, credentials)
- Regulatory requirements (GDPR, HIPAA)
- Federation — messages transiting through servers you don't control
