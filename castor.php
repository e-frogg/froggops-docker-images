<?php

declare(strict_types=1);

use Castor\Attribute\AsTask;

use function Castor\context;
use function Castor\exec;
use function Castor\io;
use function Castor\run;

#[AsTask(name: 'run', description: 'run')]
function docker_run(
    string $imageName,
    string $fopsVersion,
    string $containerCommand
): void {
    run("docker run -it --rm -v $(pwd):/app local/$imageName:$fopsVersion $containerCommand",
    context:context()->toInteractive());

}
#[AsTask(name: 'build', description: 'build')]
function docker_exec(
    string $imageName,
    string $fopsVersion
): void {
    $dockerfile = "$imageName/Dockerfile";

    if (!file_exists($dockerfile)) {
        io()->error("Error: $dockerfile does not exist");
    }

    io()->info("=== Running hadolint on $dockerfile ===");
    run("docker run --rm -i hadolint/hadolint < $dockerfile", context: context()->withAllowFailure());

    io()->info("=== Building image for $imageName ===");
    run("docker build -t local/$imageName:$fopsVersion --build-arg FOPS_IMAGE_VERSION=$fopsVersion $imageName");

    io()->info("=== Running trivy scan on local/$imageName:$fopsVersion ===");
    run(
        "docker run --rm \
        -v /tmp/trivy-cache:/tmp/trivy-cache \
        -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy image \
        --severity HIGH,CRITICAL \
        --skip-dirs /root/.composer \
        --ignore-unfixed \
        --exit-code 1 \
        --cache-dir /tmp/trivy/ \
        local/$imageName:dev || true"
    );
}
