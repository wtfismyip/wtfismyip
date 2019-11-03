FROM nginx:mainline

MAINTAINER Clint Ruoho clint@wtfismyip.com

COPY nginx /etc/nginx
COPY www /var/www/html

CMD ["nginx", "-g", "daemon off;"]
