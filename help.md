Modmail is a bot that helps users contact moderators privately. It does so by providing a thin layer on top of Discord's existing Threads feature, allowing users to send messages in a private channel, that is attached to a parent channel.

## Commands
- `/help` - Show this help message.
- `/ping` - Check if the bot is online.
- `/create-report-button` - Create a button users can click to get in touch with moderators.
  - `label: text` - The label of the button.
  - `color: color` - (Optional) The color of the button. Defaults to blue.
  - `role: role` - (Optional) The role that can use the button. Defaults to nobody.
    - **Note:** If no role is specified, moderators or other intended support will not be notified of new reports. If a non-mentionable role is specified, the bot must have the permission for mentioning all roles.

## Permissions

- **__Required Permissions__**
  - **View Channel:** The bot cannot create threads in channels it cannot view. (It can still create the button, as an interaction response)
  - **Send Messages in Threads:** Otherwise it will create an empty private thread with no participants.
  - **Create Private Threads:** ...to create *private* threads.
  - **Embed Links:** This is required to include the content submitted by the user.
- **__Optional Permissions__**
  - **Manage Threads:** To set Threads to non-invitable and extend the thread's archive timer.
  - **Mention Everyone:** To mention the role specified in `/create-report-button`.

## Getting Started
1. Set up a channel that will be used for modmail.
2. Set it up so that users can only read messages, but cannot create threads.
    - If desired, you can allow users to manually create private threads.
3. Ensure the bot has the required permissions; it should by default, but you can restrict it to only the modmail channel if you want.
4. Type up some guidelines or other information that may be pertinent to users for when they contact you.
5. Run `/create-report-button` in the channel you want the button to be in.
6. Done! Users can now click the button to contact you, and threads will be created in the modmail channel.