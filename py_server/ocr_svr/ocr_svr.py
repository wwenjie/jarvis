from fastapi import FastAPI, File, UploadFile
from fastapi.responses import JSONResponse
import easyocr
import io
from PIL import Image
import numpy as np
import os
import logging

# 配置日志
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI()

# 设置 EasyOCR 模型路径（容器内路径）
os.environ['EASYOCR_MODULE_PATH'] = '/app/.EasyOCR'

# 初始化 EasyOCR 读取器（使用非量化模型）
try:
    logger.info("正在初始化 EasyOCR 读取器...")
    reader = easyocr.Reader(
        ['ch_sim', 'en'],
        gpu=False,  # 强制使用 CPU
        # quantize=False,  # 关键参数：禁用量化
        model_storage_directory='/app/.EasyOCR',
        download_enabled=False  # 禁用下载，使用本地模型
    )
    logger.info("EasyOCR 读取器初始化成功")
except Exception as e:
    logger.error(f"EasyOCR 读取器初始化失败: {str(e)}")
    reader = None

@app.post('/ocr')
async def ocr_recognize(file: UploadFile = File(...)):
    try:
        logger.info(f"开始处理文件: {file.filename}")
        
        # 检查 EasyOCR 读取器是否可用
        if reader is None:
            return JSONResponse({"error": "EasyOCR 读取器未初始化"}, status_code=500)
        
        # 异步读文件
        logger.info("正在读取文件...")
        image_bytes = await file.read()
        logger.info(f"文件读取完成，大小: {len(image_bytes)} bytes")
        
        # 将字节流转换为 PIL 图像
        logger.info("正在转换图像格式...")
        image = Image.open(io.BytesIO(image_bytes)).convert('RGB')
        logger.info(f"图像转换完成，尺寸: {image.size}")
        
        # 如果图像太大，先resize以减少内存使用
        max_size = 1024
        if max(image.size) > max_size:
            logger.info(f"图像尺寸过大，正在resize到最大边长为{max_size}...")
            ratio = max_size / max(image.size)
            new_size = (int(image.size[0] * ratio), int(image.size[1] * ratio))
            # 使用兼容的resize方法
            try:
                image = image.resize(new_size, Image.Resampling.LANCZOS)
            except AttributeError:
                # 兼容旧版本Pillow
                image = image.resize(new_size, Image.ANTIALIAS)
            logger.info(f"图像resize完成，新尺寸: {image.size}")
        
        # 将 PIL 图像转换为 numpy 数组
        img_array = np.array(image)
        logger.info(f"图像数组转换完成，形状: {img_array.shape}")
        
        # 使用 EasyOCR 进行文字识别
        logger.info("开始OCR识别...")
        result = reader.readtext(img_array)
        logger.info(f"OCR识别完成，识别到 {len(result)} 个文本区域")
        
        # 格式化识别结果
        ocr_results = []
        for detection in result:
            bbox, text, confidence = detection
            # 确保所有数值都转换为Python原生类型
            ocr_results.append({
                "text": text,
                "confidence": float(confidence),
                "bbox": [[float(x) for x in point] for point in bbox]
            })
        
        logger.info("OCR处理完成")
        return JSONResponse({
            "success": True,
            "results": ocr_results,
            "total_texts": len(ocr_results)
        })
        
    except Exception as e:
        logger.error(f"OCR处理过程中出现错误: {str(e)}")
        import traceback
        logger.error(f"错误详情: {traceback.format_exc()}")
        return JSONResponse({"error": str(e)}, status_code=500)

@app.get("/")
def root():
    return {"msg": "EasyOCR FastAPI 文字识别服务已启动"}