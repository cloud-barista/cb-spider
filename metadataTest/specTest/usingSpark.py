from pyspark.sql import SparkSession
from pyspark.sql.functions import col, count

spark = SparkSession.builder.appName("SpecAnalysis").getOrCreate()

invalid_specs_df = spark.read.option("header", True).csv("./invalid_specs.csv")

invalid_specs_df.show(5, truncate=False)

invalid_specs_df.groupBy("InvalidReason").count().show()

connection_invalid_counts = invalid_specs_df.groupBy("ConnectionName", "InvalidReason").agg(count("*").alias("Count"))

connection_invalid_counts.orderBy(col("Count").desc()).show()