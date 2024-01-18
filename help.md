Modmail is a bot that helps users contact moderators privately.
It does so by providing a thin layer on top of Discord's existing
Threads feature, allowing users to send messages in a private channel,
that is attached to a parent channel.

## Commands
- `/help` - Show this help message.
- `/ping` - Check if the bot is online.
- `/create-report-button` - Create a button users can click to get in touch with moderators.
  - `label: text` - The label of the button.
  - `color: color` - (Optional) The color of the button. Defaults to blue.
  - `role: role` - (Optional) The role that can use the button. Defaults to nobody.
    - **Note:** If no role is specified, moderators or other intended support will not be notified of new reports.
      If a non-mentionable role is specified, the bot must have the permission for mentioning all roles.

## Permissions

- **__Required Permissions__**
  - **View Channel:** The bot cannot create threads in channels it cannot view. (It can still create the button, as an interaction response)
  - **Send Messages in Threads:** Otherwise it will create an empty private thread with no participants.
  - **Create Private Threads:** ...to create *private* threads.
  - **Embed Links:** This is required to include the content submitted by the user.
- **__Optional Permissions__**
  - **Manage Threads:** To set Threads to non-invitable and extend the thread's archive timer.
  - **Mention Everyone:** To mention the role specified in `/create-report-button`.