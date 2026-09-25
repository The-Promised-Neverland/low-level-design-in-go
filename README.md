# Low-Level Design in Go

A personal collection of LLD practice projects — mostly concurrency-heavy, written while learning Go properly.

## Before you browse

If you are mainly looking for **interview-checklist LLD** — a neat class diagram, a couple of interfaces, and a happy-path walkthrough — this repo may not be the best fit.

A few of these problems start where interviews usually stop. Especially the parking system: robots actually move cars, floors block and wake waiters, strategies reshuffle live inventory, cycles need buffer slots. That is more than "design parking lot on a whiteboard."

I am not claiming these are production systems or perfect solutions. I built them to push myself — to feel mutexes, channels, condition variables, and failure modes in code, not only in notes.

If you are here for the **love of Go's concurrency model**, feel free to open a folder, read slowly, disagree, and improve on it. That spirit is welcome here.

## What's inside

| Project | What it explores |
| --- | --- |
| [Parking_Management](./Parking_Management) | Automated multi-floor parking with robot pools, strategy switching, rearrange graphs, buffer slots for cycles |
| [Token Bucket Rate Limiter](./Token%20Bucket%20Rate%20Limiter) | Per-key token buckets, concurrent `Allow`, fair isolation across users |
| [Job Queue](./Job%20Queue) | Worker pool, retries, dead-letter queue, graceful shutdown |
| [Connection Pool](./Connection%20Pool) | Acquire / release, max open connections, idle cleanup |
| [KeyValueStore](./KeyValueStore) | Thread-safe KV with TTL and background expiry (not one timer per key) |
| [PubSub](./PubSub) | In-memory broker, bounded subscriber buffers, slow consumers |
| [LRU cache](./LRU%20cache) | Concurrent LRU get/put with eviction |
| [Snake_Ladders](./Snake_Ladders) | Classic board-game LLD (multiplayer, dice, snakes/ladders) |

Each folder has its own README with the problem statement and how to run / test.

## How I treat this repo

- Prefer **working concurrent code** over polished packaging.
- Keep problem statements honest — requirements first, ego second.
- Leave room to be wrong. Learning in public is the point.

If something helps you, or you spot a race I missed, that already made the work worthwhile.

## Language

All projects are in **Go**.
