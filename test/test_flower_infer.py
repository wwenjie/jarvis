#!/usr/bin/env python3
import os
import requests
import json
from pathlib import Path
import time

def test_flower_inference(image_dir, api_url):
    """
    测试花朵识别接口
    
    Args:
        image_dir: 图片目录路径
        api_url: API接口地址
    """
    
    # 支持的图片格式
    image_extensions = {'.jpg', '.jpeg', '.png', '.bmp', '.tiff', '.webp'}
    
    # 获取所有图片文件
    image_files = []
    for ext in image_extensions:
        image_files.extend(Path(image_dir).glob(f'*{ext}'))
        image_files.extend(Path(image_dir).glob(f'*{ext.upper()}'))
    
    if not image_files:
        print(f"在目录 {image_dir} 中没有找到图片文件")
        return
    
    print(f"找到 {len(image_files)} 个图片文件")
    print("=" * 80)
    
    # 测试每个图片
    results = []
    for i, image_path in enumerate(image_files, 1):
        print(f"[{i}/{len(image_files)}] 测试: {image_path.name}")
        
        try:
            # 准备文件
            with open(image_path, 'rb') as f:
                files = {'file': (image_path.name, f, 'image/jpeg')}
                
                # 发送请求
                start_time = time.time()
                response = requests.post(api_url, files=files, timeout=30)
                end_time = time.time()
                
                if response.status_code == 200:
                    result = response.json()
                    predicted_class = result.get('predicted_class', -1)
                    class_name = result.get('class_name', 'unknown')
                    
                    # 提取文件名中的关键词（去掉扩展名）
                    filename_keywords = image_path.stem.lower().split('_')
                    
                    print(f"  文件名: {image_path.name}")
                    print(f"  预测结果: class_id={predicted_class}, class_name='{class_name}'")
                    print(f"  响应时间: {end_time - start_time:.2f}s")
                    
                    # 检查文件名是否包含预测的花名关键词
                    class_name_words = class_name.lower().split()
                    match_score = 0
                    for word in class_name_words:
                        if any(word in keyword for keyword in filename_keywords):
                            match_score += 1
                    
                    if match_score > 0:
                        print(f"  ✓ 文件名与预测结果匹配度: {match_score}/{len(class_name_words)}")
                    else:
                        print(f"  ✗ 文件名与预测结果不匹配")
                    
                    results.append({
                        'filename': image_path.name,
                        'predicted_class': predicted_class,
                        'class_name': class_name,
                        'response_time': end_time - start_time,
                        'match_score': match_score,
                        'status': 'success'
                    })
                    
                else:
                    print(f"  ✗ 请求失败: HTTP {response.status_code}")
                    print(f"  错误信息: {response.text}")
                    results.append({
                        'filename': image_path.name,
                        'status': 'error',
                        'error': f"HTTP {response.status_code}: {response.text}"
                    })
                    
        except Exception as e:
            print(f"  ✗ 测试失败: {str(e)}")
            results.append({
                'filename': image_path.name,
                'status': 'error',
                'error': str(e)
            })
        
        print("-" * 60)
    
    # 统计结果
    print("\n" + "=" * 80)
    print("测试结果统计:")
    print(f"总文件数: {len(image_files)}")
    
    success_count = sum(1 for r in results if r['status'] == 'success')
    error_count = len(results) - success_count
    
    print(f"成功: {success_count}")
    print(f"失败: {error_count}")
    
    if success_count > 0:
        avg_response_time = sum(r['response_time'] for r in results if r['status'] == 'success') / success_count
        print(f"平均响应时间: {avg_response_time:.2f}s")
        
        # 匹配度统计
        match_scores = [r['match_score'] for r in results if r['status'] == 'success']
        perfect_matches = sum(1 for score in match_scores if score > 0)
        print(f"文件名与预测结果有匹配的文件数: {perfect_matches}/{success_count}")
    
    # 保存详细结果到文件
    output_file = "flower_inference_test_results.json"
    with open(output_file, 'w', encoding='utf-8') as f:
        json.dump(results, f, ensure_ascii=False, indent=2)
    
    print(f"\n详细结果已保存到: {output_file}")
    
    # 输出表格
    print_table(results)

def print_table(results):
    """
    按文件名排序输出表格
    """
    # 按文件名排序
    sorted_results = sorted(results, key=lambda x: x['filename'])
    
    print("\n" + "=" * 120)
    print("测试结果表格 (按文件名排序)")
    print("=" * 120)
    
    # 表头
    print(f"{'文件名':<30} {'是否匹配':<10} {'预测花名':<25} {'响应时间(s)':<12} {'状态':<10}")
    print("-" * 120)
    
    # 表格内容
    for result in sorted_results:
        filename = result['filename']
        
        if result['status'] == 'success':
            # 成功的情况
            is_match = "✓" if result['match_score'] > 0 else "✗"
            class_name = result['class_name']
            response_time = f"{result['response_time']:.2f}"
            status = "成功"
        else:
            # 失败的情况
            is_match = "✗"
            class_name = "N/A"
            response_time = "N/A"
            status = "失败"
        
        print(f"{filename:<30} {is_match:<10} {class_name:<25} {response_time:<12} {status:<10}")
    
    print("-" * 120)
    
    # 统计信息
    success_results = [r for r in sorted_results if r['status'] == 'success']
    match_results = [r for r in success_results if r['match_score'] > 0]
    
    print(f"总计: {len(sorted_results)} 个文件")
    print(f"成功: {len(success_results)} 个")
    print(f"匹配: {len(match_results)} 个")
    print(f"匹配率: {len(match_results)/len(success_results)*100:.1f}%" if success_results else "0%")

if __name__ == "__main__":
    # 配置参数
    IMAGE_DIR = "."  # 当前目录，你可以修改为实际路径
    API_URL = "http://10.1.20.9:8082/infer"  # 你的API地址
    
    print("花朵识别接口测试脚本")
    print(f"图片目录: {IMAGE_DIR}")
    print(f"API地址: {API_URL}")
    print()
    
    test_flower_inference(IMAGE_DIR, API_URL)