{
    description = "Lemonilemon's NixOS Flake";

    nixConfig = {
        accept-flake-config = true;
        substituters = [
	        "https://aseipp-nix-cache.global.ssl.fastly.net"
            "https://mirrors.ustc.edu.cn/nix-channels/store"
            "https://cache.nixos.org/"
        ];
        extra-substituters = [
            "https://nix-community.cachix.org"
        ];
        extra-trusted-public-keys = [
            "nix-community.cachix.org-1:mB9FSh9qf2dCimDSUo8Zy7bkq5CX+/rkCWyvRCYg3Fs="
        ];
    };

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
        nixosConfigurations = {
            wsl = let 
                username = "lemonilemon";
            in 
            nixpkgs.lib.nixosSystem {
                system = "x86_64-linux";
                specialArgs = {inherit inputs username;};
                modules = [
                    nixos-wsl.nixosModules.wsl
                    ./hosts/wsl
                    
                    home-manager.nixosModules.home-manager {
                        home-manager.useGlobalPkgs = true;
                        home-manager.useUserPackages = true;
                        home-manager.extraSpecialArgs = {inherit inputs username;};
                        home-manager.users.${username} = import ./home-manager;
                    }

		            ./general
                ];
            };
        };
    };
}
