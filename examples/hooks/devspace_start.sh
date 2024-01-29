#!/bin/bash
set +e  # Continue on errors

COLOR_CYAN="\033[0;36m"
COLOR_RESET="\033[0m"

if [ ! -f "/vcluster/syncer" ]; then
  echo "Downloading vCluster syncer..."
  mkdir -p /vcluster
  curl -L -o /vcluster/syncer "https://github.com/loft-sh/vcluster/releases/download/v0.19.0-alpha.3/syncer-linux-$(go env GOARCH)"
  chmod +x /vcluster/syncer
  echo "Successfully downloaded syncer"
fi

RUN_CMD="go build -mod vendor -o plugin main.go && /vcluster/syncer start"

echo -e "${COLOR_CYAN}
   ____              ____
  |  _ \  _____   __/ ___| _ __   __ _  ___ ___
  | | | |/ _ \ \ / /\___ \| '_ \ / _\` |/ __/ _ \\
  | |_| |  __/\ V /  ___) | |_) | (_| | (_|  __/
  |____/ \___| \_/  |____/| .__/ \__,_|\___\___|
                          |_|
${COLOR_RESET}
Welcome to your development container!
This is how you can work with it:
- Run \`${COLOR_CYAN}${RUN_CMD}${COLOR_RESET}\` to start the plugin
- ${COLOR_CYAN}Files will be synchronized${COLOR_RESET} between your local machine and this container

${COLOR_CYAN}TIP:${COLOR_RESET} hit an up arrow on your keyboard to find the commands mentioned above :)
"
# add useful commands to the history for convenience
export HISTFILE=/tmp/.bash_history
history -s $RUN_CMD
history -a

bash