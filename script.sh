# env variables:
echo "POM_PATH: ${PLUGIN_POM_PATH}"
echo "DOCKER_IMAGE_NAME: ${PLUGIN_DOCKER_IMAGE_NAME}"

# Check if docker image name is set
if [ -z "$PLUGIN_DOCKER_IMAGE_NAME" ]; then
  echo "Docker image name is not set"
  exit 1
fi

# Check if pom path is set
if [ -z "$PLUGIN_POM_PATH" ]; then
  echo "POM path is not set"
  exit 1
fi

#  check dockerhub credentials
if [ -z "$PLUGIN_DOCKERHUB_USERNAME" ]; then
  echo "Dockerhub username is not set"
  exit 1
fi

if [ -z "$PLUGIN_DOCKERHUB_PAT" ]; then
  echo "Dockerhub password is not set"
  exit 1
fi


# find the version of the project
POM_VERSION=$(mvn -f $PLUGIN_POM_PATH/pom.xml help:evaluate -Dexpression=project.version -q -DforceStdout) || exit 1

echo "Found version ${POM_VERSION}"

# export pom version
export MAVEN_PROJECT_VERSION=${POM_VERSION}

# Build docker image with the above version
docker build --build-arg MAVEN_PROJECT_VERSION=${MAVEN_PROJECT_VERSION} -t ${PLUGIN_DOCKER_IMAGE_NAME}:latest . || exit 1

docker tag ${PLUGIN_DOCKER_IMAGE_NAME}:latest ${PLUGIN_DOCKERHUB_USERNAME}/${PLUGIN_DOCKER_IMAGE_NAME}:latest || exit 1

# Push docker image to dockerhub
docker login -u ${PLUGIN_DOCKERHUB_USERNAME} -p ${PLUGIN_DOCKERHUB_PAT} || exit 1
docker push ${PLUGIN_DOCKERHUB_USERNAME}/${PLUGIN_DOCKER_IMAGE_NAME}:latest || exit 1

echo "Successfully pushed docker image to dockerhub - ${PLUGIN_DOCKERHUB_USERNAME}/${PLUGIN_DOCKER_IMAGE_NAME}:latest"
