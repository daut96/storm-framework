# MIT License.
# Copyright (c) 2026 Storm Framework

# See LICENSE file in the project root for full license information.


import app.base.config_ui as ui


# Display help to make it easier for users to see what commands are available.
# Without this the user is confused about what commands are in storm.
def execute(args, context):
    ui.show_help()
    return context
