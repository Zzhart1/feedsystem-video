#
# 前端多阶段构建：node 构建 → nginx 伺服
# 构建：  docker build -t feedsystem-frontend .
# 运行：  docker run -d --name feedsystem-frontend -p 5173:80 feedsystem-frontend
#

FROM node:22-alpine AS build
WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY . ./
RUN npm run build

FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
