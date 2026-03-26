# Enhanced Notification System

The marchat client now features a comprehensive notification system that goes beyond simple bell notifications.

## Features

### 1. Multiple Notification Modes

- **None**: All notifications disabled
- **Bell**: Terminal bell only (original behavior)
- **Desktop**: Platform-specific desktop notifications for all messages
- **Both**: Bell + desktop notifications for all messages

**Note:** When you set mode to `desktop` or `both`, it automatically enables notifications for **all messages**. If you only want notifications for @mentions, enable desktop mode and then use `:bell-mention` to restrict to mentions only.

### 2. Notification Levels

Messages are classified by priority:
- **Info**: Regular chat messages
- **Mention**: When someone @mentions you
- **DM**: Direct messages (use `:dm <user> <message>` or `:dm <user>` to toggle DM mode)
- **Urgent**: System alerts and critical messages

### 3. Desktop Notifications

Cross-platform support:
- **Linux**: Uses `notify-send` (libnotify)
- **macOS**: Uses `osascript` with AppleScript
- **Windows**: Uses PowerShell toast notifications

Desktop notifications automatically:
- Detect platform support at startup
- Truncate long messages (100 chars max)
- Rate limit to prevent spam (2 seconds between notifications)
- Run asynchronously to avoid blocking

### 4. Smart Features

#### Quiet Hours
Disable notifications during specified hours:
```
:quiet 22 8        # Quiet from 10 PM to 8 AM
:quiet-off         # Disable quiet hours
```

Supports overnight ranges (e.g., 22:00 to 08:00).

#### Focus Mode
Temporarily disable all notifications:
```
:focus             # Default 30 minutes
:focus 1h          # 1 hour focus mode
:focus 2h30m       # 2 hours 30 minutes
:focus-off         # Disable focus mode
```

#### Mention-Only Mode
Only notify when someone @mentions you:
```
:bell-mention      # Toggle mention-only mode
```

## Commands

### Basic Notification Commands

| Command | Description |
|---------|-------------|
| `:bell` | Toggle bell notifications on/off |
| `:bell-mention` | Toggle mention-only bell mode |

### Enhanced Notification Commands

| Command | Description |
|---------|-------------|
| `:notify-mode <mode>` | Set mode: `none`, `bell`, `desktop`, or `both` |
| `:notify-desktop` | Toggle desktop notifications |
| `:notify-status` | Show current notification settings |
| `:quiet <start> <end>` | Enable quiet hours (24h format) |
| `:quiet-off` | Disable quiet hours |
| `:focus [duration]` | Enable focus mode (default 30m) |
| `:focus-off` | Disable focus mode |

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Alt+N` | Toggle desktop notifications |

## Configuration

Notification settings are automatically saved to your client `config.json` (same directory as profiles and themes: per-user app data, or `MARCHAT_CONFIG_DIR` — see [README.md](README.md) and [ARCHITECTURE.md](ARCHITECTURE.md)). Run `marchat-client -doctor` to confirm the resolved path. Example shape:

```json
{
  "notification_mode": "bell",
  "desktop_notifications": false,
  "desktop_on_mention": true,
  "desktop_on_dm": true,
  "desktop_on_all": false,
  "quiet_hours_enabled": false,
  "quiet_hours_start": 22,
  "quiet_hours_end": 8,
  "enable_bell": true,
  "bell_on_mention": false
}
```

## Usage Examples

### Setup Desktop Notifications

```bash
# Enable desktop notifications
:notify-mode desktop

# Or enable both bell and desktop
:notify-mode both

# Toggle desktop notifications with hotkey
Alt+N
```

### Configure Quiet Hours

```bash
# No notifications from 10 PM to 8 AM
:quiet 22 8

# No notifications during work hours (9 AM - 5 PM)
:quiet 9 17

# Disable quiet hours
:quiet-off
```

### Use Focus Mode

```bash
# Quick focus (30 minutes)
:focus

# Extended focus session
:focus 2h

# Exit focus mode early
:focus-off
```

### Check Notification Status

```bash
:notify-status

# Output example:
# Mode: both | Bell: true (mention-only: false) | Desktop: true (supported: true) | Quiet hours: 22:00 - 08:00
```

## Platform Requirements

### Linux
- **Desktop notifications**: Requires `libnotify` (notify-send)
  ```bash
  # Ubuntu/Debian
  sudo apt install libnotify-bin
  
  # Fedora/RHEL
  sudo dnf install libnotify
  
  # Arch
  sudo pacman -S libnotify
  ```

### macOS
- **Desktop notifications**: Built-in `osascript` (no installation needed)

### Windows
- **Desktop notifications**: Built-in PowerShell (no installation needed)

## Backward Compatibility

The new system maintains full backward compatibility:
- Existing `:bell` and `:bell-mention` commands work as before
- Legacy config settings (`enable_bell`, `bell_on_mention`) are preserved
- Default mode is `bell` (original behavior)

## Implementation Details

### Rate Limiting
- **Bell**: Minimum 500ms between notifications
- **Desktop**: Minimum 2 seconds between notifications

### Notification Logic

1. Check if notifications are enabled (mode != none)
2. Check quiet hours and focus mode
3. Determine notification level based on message content
4. Apply user preferences (mention-only, desktop-only, etc.)
5. Send appropriate notifications (bell and/or desktop)

### Thread Safety

The NotificationManager uses mutex locks to ensure thread-safe operation:
- Safe for concurrent notification calls
- Configuration updates are atomic

## Testing

Run tests with:
```bash
go test ./client -v -run TestNotificationManager
```

The test suite covers:
- Manager initialization
- Mode switching
- Bell and desktop toggle
- Quiet hours configuration
- Focus mode
- Notification sending (all levels)

## Future Enhancements

Potential future additions:
- Per-user notification rules
- Custom notification sounds
- Notification history/log
- Priority-based notification filtering
- Integration with system Do Not Disturb
- Notification action buttons (reply, dismiss, etc.)

