//! Go Template LSP extension for Zed.
//!
//! Downloads and manages the go-template-lsp binary from GitHub releases.

use std::fs;
use zed_extension_api::{self as zed, LanguageServerId, Result, Worktree};

/// The GitHub repository owner/name for downloading releases.
const GITHUB_REPO: &str = "toba/go-template-lsp";

/// The name of the LSP binary.
const BINARY_NAME: &str = "go-template-lsp";

struct GoTemplateLspExtension {
    cached_binary_path: Option<String>,
}

impl GoTemplateLspExtension {
    /// Returns the path to the LSP binary, downloading it if necessary.
    fn language_server_binary_path(
        &mut self,
        language_server_id: &LanguageServerId,
        worktree: &Worktree,
    ) -> Result<String> {
        // Check cache first
        if let Some(path) = &self.cached_binary_path {
            if fs::metadata(path).is_ok() {
                return Ok(path.clone());
            }
        }

        // Check for existing binary in worktree (for dev extensions)
        let worktree_root = worktree.root_path();
        let dev_binary_path = format!("{}/{}", worktree_root, BINARY_NAME);
        if fs::metadata(&dev_binary_path).is_ok() {
            self.cached_binary_path = Some(dev_binary_path.clone());
            return Ok(dev_binary_path);
        }

        // Download from GitHub releases
        zed::set_language_server_installation_status(
            language_server_id,
            &zed::LanguageServerInstallationStatus::CheckingForUpdate,
        );

        let release = zed::latest_github_release(
            GITHUB_REPO,
            zed::GithubReleaseOptions {
                require_assets: true,
                pre_release: false,
            },
        )?;

        let (platform, arch) = zed::current_platform();
        let asset_name = format!(
            "go-template-lsp_{os}_{arch}.{ext}",
            os = match platform {
                zed::Os::Mac => "darwin",
                zed::Os::Linux => "linux",
                zed::Os::Windows => "windows",
            },
            arch = match arch {
                zed::Architecture::Aarch64 => "arm64",
                zed::Architecture::X8664 => "amd64",
                zed::Architecture::X86 => "386",
            },
            ext = match platform {
                zed::Os::Windows => "zip",
                _ => "tar.gz",
            }
        );

        let asset = release
            .assets
            .iter()
            .find(|a| a.name == asset_name)
            .ok_or_else(|| format!("no asset found matching {}", asset_name))?;

        let version_dir = format!("{}-{}", BINARY_NAME, release.version);
        let binary_path = format!(
            "{}/{}{}",
            version_dir,
            BINARY_NAME,
            match platform {
                zed::Os::Windows => ".exe",
                _ => "",
            }
        );

        // Check if already downloaded
        if fs::metadata(&binary_path).is_err() {
            zed::set_language_server_installation_status(
                language_server_id,
                &zed::LanguageServerInstallationStatus::Downloading,
            );

            zed::download_file(
                &asset.download_url,
                &version_dir,
                match platform {
                    zed::Os::Windows => zed::DownloadedFileType::Zip,
                    _ => zed::DownloadedFileType::GzipTar,
                },
            )
            .map_err(|e| format!("failed to download file: {e}"))?;

            // Make binary executable on Unix
            zed::make_file_executable(&binary_path)?;
        }

        self.cached_binary_path = Some(binary_path.clone());
        Ok(binary_path)
    }
}

impl zed::Extension for GoTemplateLspExtension {
    fn new() -> Self {
        Self {
            cached_binary_path: None,
        }
    }

    fn language_server_command(
        &mut self,
        language_server_id: &LanguageServerId,
        worktree: &Worktree,
    ) -> Result<zed::Command> {
        let binary_path = self.language_server_binary_path(language_server_id, worktree)?;

        Ok(zed::Command {
            command: binary_path,
            args: vec![],
            env: worktree.shell_env(),
        })
    }
}

zed::register_extension!(GoTemplateLspExtension);
