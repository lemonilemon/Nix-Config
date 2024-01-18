{
    description = "Lemonilemon's NixOS Flake";


    inputs = {
        nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
	    nixos-hardware.url = "github:NixOS/nixos-hardware/master";
        nixos-wsl = {
            url = "github:nix-community/NixOS-WSL";
            inputs.nixpkgs.follows = "nixpkgs";
        };
        home-manager = {
            url = "github:nix-community/home-manager";
            inputs.nixpkgs.follows = "nixpkgs";
        };
    };
    outputs = inputs@ { 
        self,
        nixpkgs, 
        nixos-wsl,
        nixos-hardware,
        home-manager,
        ... 
    }: {
        nixosConfigurations = let 
            username = "lemonilemon";
        in {
            wsl = let 
                hostname = "wsl";
            in 
            nixpkgs.lib.nixosSystem {
                system = "x86_64-linux";
                specialArgs = {inherit inputs username hostname;};
                modules = [
                    # nixos-wsl
                    nixos-wsl.nixosModules.wsl
                    
                    # home-manager
                    home-manager.nixosModules.home-manager {
                        home-manager.useGlobalPkgs = true;
                        home-manager.useUserPackages = true;
                        home-manager.extraSpecialArgs = {
                            inherit inputs username hostname;
                            WSL = true;
                        };
                        home-manager.users.${username} = import ./home-manager;
                    }

                    # customized settings
                    ./hosts/wsl
		            ./general
                ];
            };
        };
    };
}
