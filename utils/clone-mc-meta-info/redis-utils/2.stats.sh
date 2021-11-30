sudo docker stats --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" redis-for-mini
