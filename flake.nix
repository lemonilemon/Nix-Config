{
    description = "Lemonilemon's Nix Flake";


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
        systems.url = "github:nix-systems/default";
    };
    outputs = inputs@ { 
        self,
        nixpkgs, 
        nixos-wsl,
        nixos-hardware,
        home-manager,
        systems,
        ... 
    }: let 
        eachSystem = nixpkgs.lib.genAttrs (import systems);
        username = "lemonilemon";
        pkgs = eachSystem (system: nixpkgs.legacyPackages.${system});
    in {
        formatter = eachSystem (system: nixpkgs.legacyPackages.${system}.nixpkgs-fmt);

        # For non-NixOS
        homeManagerConfiguration.${username} = home-manager.lib.homeManagerConfiguration {
            inherit pkgs;
            modules = [ 
                ./home-manager 
                ./general
            ];
            extraSpecialArgs = { inherit inputs username; };
        };

        # For NixOS
        nixosConfigurations = let 
            hostname = "SpaceNix";
        in {
            NixOS-wsl = let  
                sys = "NixOS-wsl";
            in nixpkgs.lib.nixosSystem {
                system = "x86_64-linux";
                specialArgs = { inherit inputs username hostname; };
                modules = [
                    # nixos-wsl
                    nixos-wsl.nixosModules.wsl
                    
                    # home-manager
                    home-manager.nixosModules.home-manager {
                        home-manager.useGlobalPkgs = true;
                        home-manager.useUserPackages = true;
                        home-manager.extraSpecialArgs = {
                            inherit inputs username hostname sys;
                            WSL = true;
                            GUI = false;
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
