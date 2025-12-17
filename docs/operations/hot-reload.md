# Config Hot Reload

> Implemented: Detect -> validate -> atomic swap -> rollback.

The gateway watches the configuration file for changes (polling every 5 seconds).
When a change is detected:
1. **Detect**: File modification time change.
2. **Validate**: The new configuration is loaded and parsed. If parsing fails, the reload is aborted (Rollback/Ignore).
3. **Atomic Swap**: If valid, the internal state (routes, services, balancers) is atomically swapped using a mutex.
4. **Rollback**: Implicitly handled by not swapping if validation fails.

## Remote-Config
> TODO: (Unreleased) Pull/push model sketch and minimal safeguards.
