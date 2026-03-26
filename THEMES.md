# Custom Themes Guide

Marchat supports custom color schemes through JSON configuration files. This allows you to personalize the appearance of the chat client to match your preferences.

## Quick Start

1. Copy `themes.example.json` to `themes.json`. The client loads, in order: `./themes.json` in the **current working directory**, then `themes.json` in the **client configuration directory** (same folder as `config.json` and profiles: per-user app data on your OS, or `MARCHAT_CONFIG_DIR`). Run `marchat-client -doctor` to see the resolved path. Typical per-user paths: `~/.config/marchat/themes.json` (Linux), `~/Library/Application Support/marchat/themes.json` (macOS), `%APPDATA%\marchat\themes.json` (Windows).

2. Edit `themes.json` to define your custom themes

3. Use `:theme <name>` to switch to your custom theme, or press `Ctrl+T` to cycle through all themes

## Built-in Themes

Marchat comes with 4 built-in themes:

- **system** - Uses your terminal's default colors (no custom colors)
- **patriot** - American patriotic theme with red, white, and blue
- **retro** - Retro terminal theme with orange, green, and yellow
- **modern** - Modern dark theme with blue-gray tones

## Custom Theme Format

```json
{
  "theme-name": {
    "name": "Display Name",
    "description": "Optional description of the theme",
    "colors": {
      "user": "#4F8EF7",
      "time": "#A0A0A0",
      "message": "#E0E0E0",
      "banner": "#FF5F5F",
      "box_border": "#4F8EF7",
      "mention": "#FFD700",
      "hyperlink": "#4A9EFF",
      "user_list_border": "#4F8EF7",
      "me": "#4F8EF7",
      "other": "#AAAAAA",
      "background": "#181C24",
      "header_bg": "#4F8EF7",
      "header_fg": "#FFFFFF",
      "footer_bg": "#181C24",
      "footer_fg": "#4F8EF7",
      "input_bg": "#23272E",
      "input_fg": "#E0E0E0",
      "help_overlay_bg": "#1a1a1a",
      "help_overlay_fg": "#FFFFFF",
      "help_overlay_border": "#FFFFFF",
      "help_title": "#FFD700"
    }
  }
}
```

## Color Properties

| Property | Description |
|----------|-------------|
| `user` | Username color in messages |
| `time` | Timestamp color |
| `message` | Message text color |
| `banner` | Banner/notification color |
| `box_border` | Chat viewport border color |
| `mention` | @mention highlight color |
| `hyperlink` | URL link color |
| `user_list_border` | User list panel border color |
| `me` | Your own username in user list |
| `other` | Other users in user list |
| `background` | Main background color |
| `header_bg` | Header background color |
| `header_fg` | Header text color |
| `footer_bg` | Footer background color |
| `footer_fg` | Footer text color |
| `input_bg` | Input area background |
| `input_fg` | Input text color |
| `help_overlay_bg` | Help menu background |
| `help_overlay_fg` | Help menu text |
| `help_overlay_border` | Help menu border |
| `help_title` | Help menu title color |

## Using Custom Themes

### List Available Themes
```
:themes
```

This will show all available themes (built-in + custom) with descriptions.

### Switch to a Theme
```
:theme dracula
```

### Cycle Through Themes
Press `Ctrl+T` to cycle through all available themes (built-in and custom).

### Set Default Theme
Use the interactive configuration or edit your config file:

```json
{
  "theme": "dracula",
  ...
}
```

## Example Themes

The `themes.example.json` file includes several ready-to-use themes:

- **custom-dark** - A customizable dark theme template
- **cyberpunk** - Vibrant neon colors inspired by cyberpunk aesthetics
- **forest** - Calming green tones inspired by nature
- **dracula** - Based on the popular Dracula color scheme
- **solarized-dark** - Based on the Solarized Dark palette
- **nord** - Based on the Nord color scheme

## Creating Your Own Theme

1. Start with one of the example themes
2. Modify the colors to your preference
3. Colors should be in hex format: `#RRGGBB`
4. Test your theme with `:theme your-theme-name`
5. Save it permanently by keeping it in `themes.json`

## Tips

- Use a color picker tool to find hex codes for colors you like
- Test your theme in different lighting conditions
- Consider accessibility - ensure good contrast between text and backgrounds
- You can define multiple themes and switch between them easily
- Custom themes persist across client restarts

## Troubleshooting

**Theme not loading?**
- Check that your `themes.json` is valid JSON (use a JSON validator)
- Ensure the file is in the correct location
- Check the client debug log at `~/.config/marchat/marchat-client-debug.log`

**Colors look wrong?**
- Verify hex codes are in the format `#RRGGBB` (6 hex digits)
- Some terminals may have limited color support
- Try a different terminal emulator if colors don't display correctly

**Theme not appearing in list?**
- Use `:themes` to see all available themes
- Check that your theme name doesn't conflict with built-in themes
- Restart the client after modifying `themes.json`

