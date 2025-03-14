Vagrant.configure("2") do |config|
    config.vm.box = "digital_ocean"
    config.ssh.private_key_path = "~/.ssh/do_ssh_key"
  
    config.vm.synced_folder "minitwit/remote_files", "/minitwit", type: "rsync"
    config.vm.synced_folder '.', '/vagrant', disabled: true

  
    config.vm.define "main" do |main|
      main.vm.provider :digital_ocean do |provider|
        provider.token = ENV['DIGITAL_OCEAN_TOKEN']
        provider.ssh_key_name = "do_ssh_key"
        provider.name = "minitwit-main"
        provider.region = "fra1"
        provider.size = "s-1vcpu-1gb"
        provider.image = "ubuntu-22-04-x64"
        provider.private_networking = false
        provider.ipv6 = false
      end
  
      # Set Docker credentials as environment variables
      main.vm.provision "shell", inline: 'echo "export DOCKER_USERNAME=' + "'" + ENV["DOCKER_USERNAME"] + "'" + '" >> ~/.bash_profile'
      main.vm.provision "shell", inline: 'echo "export DOCKER_PASSWORD=' + "'" + ENV["DOCKER_PASSWORD"] + "'" + '" >> ~/.bash_profile'
      
      main.vm.provision "shell", inline: <<-SHELL
        # Clean up any existing package manager locks
        sudo killall apt apt-get || true
        sudo rm -f /var/lib/dpkg/lock-frontend
  
        # Install Docker using official script
        curl -fsSL https://get.docker.com -o get-docker.sh
        sudo sh get-docker.sh
  
        # Configure Docker service
        sudo systemctl enable docker
        sudo systemctl start docker
  
        # Install Docker Compose
        sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
        sudo chmod +x /usr/local/bin/docker-compose
  
        # Verify installations
        docker --version
        docker-compose --version
  
        # Configure Docker Hub authentication
        echo "Logging in to Docker Hub..."
        echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
  
        # Deploy application
        cd /minitwit
        docker-compose pull
        docker-compose up -d
  
        # Display running containers
        docker ps
      SHELL

       
        main.vm.provision "deploy", type: "shell", inline: <<-SHELL
        
        echo "Logging in to Docker Hub..."
        echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin

        cd /minitwit
        echo "Pulling latest images..."
        docker-compose pull
        
        echo "Restarting services..."
        docker-compose up -d

        echo "Current running containers:"
        docker ps
      SHELL
    end
  end